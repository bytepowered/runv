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
	"github.com/bytepowered/runv/assert"
	"github.com/sirupsen/logrus"
)

type Options struct {
	StartupTimeout  time.Duration
	ServeTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type Application struct {
	prehooks  []func() error
	posthooks []func() error
	initables []Initable
	startups  []Startup
	shutdowns []Shutdown
	servable  Servable
	objects   []interface{}
}

var (
	app = &Application{
		initables: make([]Initable, 0, 4),
		startups:  make([]Startup, 0, 4),
		shutdowns: make([]Shutdown, 0, 4),
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
	assert.MustNotNil(activeobj, "add a nil active-object")
	if dis, ok := activeobj.(Disabled); ok {
		if reason, is := dis.Disabled(); is {
			Log().Infof("active-object is DISABLED, object: %T, reason: %s", activeobj, reason)
			return
		}
	}
	if act, ok := activeobj.(Activable); ok && !act.Active() {
		Log().Infof("active-object is NOT-ACTIVE, object: %T, reason: inactive", activeobj)
		return
	}
	containerd.Register(activeobj)
	AddStateObject(activeobj)
}

func AddStateObject(object interface{}) {
	assert.MustNotNil(object, "add a nil state-object")
	if init, ok := object.(Initable); ok {
		app.initables = append(app.initables, init)
	}
	if up, ok := object.(Startup); ok {
		app.startups = append(app.startups, up)
	}
	if down, ok := object.(Shutdown); ok {
		app.shutdowns = append(app.shutdowns, down)
	}
	if serv, ok := object.(Servable); ok {
		assert.MustNil(app.servable, fmt.Sprintf("duplicated servable object, exists: %T, tobe: %T", app.servable, serv))
		app.servable = serv
	}
	app.objects = append(app.objects, object)
}

func RunV() {
	DoRunV(Options{
		StartupTimeout:  time.Second * 10,
		ServeTimeout:    time.Second * 10,
		ShutdownTimeout: time.Second * 10,
	})
}

func DoRunV(opts Options) {
	// prepare hooks
	for _, prehook := range app.prehooks {
		if err := prehook(); err != nil {
			Log().Fatalf("pre-hook failed, err: %s", err)
		}
	}
	// resolve deps
	for _, obj := range app.objects {
		containerd.Resolve(obj)
	}
	// init
	sort.Sort(initiables(app.initables))
	for _, obj := range app.initables {
		if err := obj.OnInit(); err != nil {
			Log().Fatalf("init failed, err: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer shutdown(goctx, opts.ShutdownTimeout)
	// startup
	if err := startup(goctx, opts.StartupTimeout); err != nil {
		Log().Fatalf("startup failed, err: %s", err)
	}
	// serve
	if app.servable != nil {
		if err := app.servable.Serve(goctx); err != nil {
			Log().Fatalf("serve failed, err: %s", err)
		}
	}
	// post hook
	for _, posthook := range app.posthooks {
		if err := posthook(); err != nil {
			Log().Fatalf("post-hook failed, err: %s", err)
		}
	}
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
	sort.Sort(shutdowns(app.shutdowns))
	doshutdown := func(obj Shutdown) error {
		ctx, cancel := context.WithTimeout(goctx, timeout)
		defer cancel()
		return metric(ctx, fmt.Sprintf("[%T] shutdown...", obj), obj.Shutdown)
	}
	for _, obj := range app.shutdowns {
		if err := doshutdown(obj); err != nil {
			Log().Errorf("shutdown error: %s", err)
		}
	}
}

func startup(goctx context.Context, timeout time.Duration) error {
	sort.Sort(startups(app.startups))
	startup0 := func(obj Startup) error {
		ctx, cancel := context.WithTimeout(goctx, timeout)
		defer cancel()
		return metric(ctx, fmt.Sprintf("[%T] startup...", obj), obj.Startup)
	}
	for _, obj := range app.startups {
		if err := startup0(obj); err != nil {
			return fmt.Errorf("[%T] startup error: %s", obj, err)
		}
	}
	return nil
}

func metric(ctx context.Context, name string, step func(ctx context.Context) error) error {
	defer func(t time.Time) {
		Log().Debugf("%s elspaed: %s", name, time.Since(t))
	}(time.Now())
	return step(ctx)
}
