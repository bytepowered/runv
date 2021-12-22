package runv

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

import (
	"github.com/sirupsen/logrus"
)

type Options struct {
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
}

type application struct {
	prehooks  []func() error
	posthooks []func() error
	initables []Initable
	startups  []Startup
	shutdowns []Shutdown
	servables []Servable
	objects   []interface{}
}

var (
	app = &application{
		initables: make([]Initable, 0, 4),
		startups:  make([]Startup, 0, 4),
		shutdowns: make([]Shutdown, 0, 4),
		servables: make([]Servable, 0, 4),
		prehooks:  make([]func() error, 0, 4),
		posthooks: make([]func() error, 0, 4),
	}
	signalf = func() <-chan os.Signal {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		return sig
	}
	containerd = NewContainerd()
	logger     = NewJSONLogger()
)

func init() {
	containerd.AddHook(func(container *Containerd, _ interface{}) {
		container.Resolve(app)
	})
}

func Add(obj interface{}) {
	AddActiveObject(obj)
}

func AddActiveObject(activeobj interface{}) {
	AssertNNil(activeobj, "add a nil active-object")
	if dis, ok := activeobj.(Disabled); ok {
		if reason, is := dis.Disabled(); is {
			xlog().Infof("active-object is DISABLED, object: %T, reason: %s", activeobj, reason)
			return
		}
	}
	if act, ok := activeobj.(Activable); ok && !act.Active() {
		xlog().Infof("active-object is NOT-ACTIVE, object: %T, reason: inactive", activeobj)
		return
	}
	containerd.Register(activeobj)
	AddStateObject(activeobj)
}

func AddStateObject(stateobj interface{}) {
	AssertNNil(stateobj, "add a nil state-object")
	if init, ok := stateobj.(Initable); ok {
		app.initables = append(app.initables, init)
	}
	if up, ok := stateobj.(Startup); ok {
		app.startups = append(app.startups, up)
	}
	if down, ok := stateobj.(Shutdown); ok {
		app.shutdowns = append(app.shutdowns, down)
	}
	if serv, ok := stateobj.(Servable); ok {
		app.servables = append(app.servables, serv)
	}
	app.objects = append(app.objects, stateobj)
}

func RunV() {
	DoRunV(Options{
		StartupTimeout:  time.Second * 10,
		ShutdownTimeout: time.Second * 10,
	})
}

func DoRunV(opts Options) {
	// prepare hooks
	for _, prehook := range app.prehooks {
		if err := prehook(); err != nil {
			xlog().Fatalf("app: pre-hook error: %s", err)
		}
	}
	// resolve deps
	for _, obj := range app.objects {
		containerd.Resolve(obj)
	}
	xlog().Infof("init")
	// init
	sort.Sort(initiables(app.initables))
	for _, obj := range app.initables {
		if err := obj.OnInit(); err != nil {
			xlog().Fatalf("init error: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer shutdown(goctx, opts.ShutdownTimeout)
	// startup
	if err := startup(goctx, opts.StartupTimeout); err != nil {
		xlog().Fatalf("startup, error: %s", err)
	}
	// serve with setup
	if err := serve(goctx); err != nil {
		xlog().Fatalf("serve, error: %s", err)
	}
	// post hook
	for _, posthook := range app.posthooks {
		if err := posthook(); err != nil {
			xlog().Fatalf("post-hook error: %s", err)
		}
	}
	xlog().Infof("run-v SUCCESS!")
	<-signalf()
}

func SetLogger(l *logrus.Logger) {
	logger = l
}

func Log() *logrus.Logger {
	return logger
}

func SetSignals(aaf func() <-chan os.Signal) {
	signalf = aaf
}

func AddPreHook(hook func() error) {
	app.prehooks = append(app.prehooks, hook)
}

func AddPostHook(hook func() error) {
	app.posthooks = append(app.posthooks, hook)
}

func Container() *Containerd {
	return containerd
}

func NewJSONLogger() *logrus.Logger {
	return &logrus.Logger{
		Out:          os.Stderr,
		Formatter:    new(logrus.JSONFormatter),
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
}

func shutdown(goctx context.Context, timeout time.Duration) {
	defer xlog().Infof("terminaled")
	sort.Sort(shutdowns(app.shutdowns))
	doshutdown := func(obj Shutdown) error {
		newctx, cancel := context.WithTimeout(goctx, timeout)
		defer cancel()
		ctx := NewContextV(newctx, nil)
		return metricStd(ctx, fmt.Sprintf("[%T] shutdown...", obj), obj.Shutdown)
	}
	for _, obj := range app.shutdowns {
		if err := doshutdown(obj); err != nil {
			xlog().Errorf("shutdown error: %s", err)
		}
	}
}

func startup(goctx context.Context, timeout time.Duration) error {
	xlog().Infof("startup")
	sort.Sort(startups(app.startups))
	startup0 := func(obj Startup) error {
		newctx, cancel := context.WithTimeout(goctx, timeout)
		defer cancel()
		ctx := NewContextV(newctx, nil)
		return metricStd(ctx, fmt.Sprintf("[%T] startup...", obj), obj.Startup)
	}
	for _, obj := range app.startups {
		if err := startup0(obj); err != nil {
			return fmt.Errorf("[%T] startup error: %s", obj, err)
		}
	}
	return nil
}

func serve(goctx context.Context) error {
	xlog().Infof("serve")
	sort.Sort(servables(app.servables))
	for _, state := range app.servables {
		ctx := state.Setup(goctx)
		err := metricExt(ctx, fmt.Sprintf("[%T] serve...", state), state.Serve)
		if err != nil {
			return fmt.Errorf("[%T] serve error: %w", state, err)
		}
	}
	return nil
}

func xlog() *logrus.Entry {
	return logger.WithField("app", "runv.app")
}

func metricExt(ctx Context, name string, step func(ctx Context) error) error {
	defer func(t time.Time) {
		Log().Infof("%s elspaed: %s", name, time.Since(t))
	}(time.Now())
	return step(ctx)
}

func metricStd(ctx Context, name string, step func(ctx context.Context) error) error {
	return metricExt(ctx, name, func(ctx Context) error {
		return step(ctx)
	})
}
