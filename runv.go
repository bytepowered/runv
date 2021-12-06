package runv

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

import (
	"github.com/sirupsen/logrus"
)

type application struct {
	prehooks  []func() error
	posthooks []func() error
	initables []Initable
	startups  []Startup
	shutdown  []Shutdown
	servables []Servable
	objects   []interface{}
}

var (
	app = &application{
		initables: make([]Initable, 0, 4),
		startups:  make([]Startup, 0, 4),
		shutdown:  make([]Shutdown, 0, 4),
		servables: make([]Servable, 0, 4),
		prehooks:  make([]func() error, 0, 4),
		posthooks: make([]func() error, 0, 4),
	}
	signalf = func() <-chan os.Signal {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
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
	AssertNNil(activeobj, "app: add a nil active-object")
	if dis, ok := activeobj.(Disabled); ok {
		if reason, is := dis.Disabled(); is {
			logger.Infof("active-object is DISABLED, object: %T, reason: %s", activeobj, reason)
			return
		}
	}
	if act, ok := activeobj.(Activable); ok && !act.Active() {
		logger.Infof("active-object is NOT-ACTIVE, object: %T, reason: inactive", activeobj)
		return
	}
	containerd.Register(activeobj)
	AddStateObject(activeobj)
}

func AddStateObject(stateobj interface{}) {
	AssertNNil(stateobj, "app: add a nil state-object")
	if init, ok := stateobj.(Initable); ok {
		app.initables = append(app.initables, init)
	}
	if up, ok := stateobj.(Startup); ok {
		app.startups = append(app.startups, up)
	}
	if down, ok := stateobj.(Shutdown); ok {
		app.shutdown = append(app.shutdown, down)
	}
	if serv, ok := stateobj.(Servable); ok {
		app.servables = append(app.servables, serv)
	}
	app.objects = append(app.objects, stateobj)
}

func RunV() {
	// prepare hooks
	for _, prehook := range app.prehooks {
		if err := prehook(); err != nil {
			logger.Fatalf("app: pre-hook error: %s", err)
		}
	}
	// resolve deps
	for _, obj := range app.objects {
		containerd.Resolve(obj)
	}
	logger.Infof("app: init")
	// init
	for _, obj := range app.initables {
		if err := obj.OnInit(); err != nil {
			logger.Fatalf("app: init error: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer shutdown(goctx)
	// startup
	if err := startup(goctx); err != nil {
		logger.Fatalf("app: startup, error: %s", err)
	}
	// serve with setup
	if err := serve(goctx); err != nil {
		logger.Fatalf("app: serve, error: %s", err)
	}
	// post hook
	for _, posthook := range app.posthooks {
		if err := posthook(); err != nil {
			logger.Fatalf("app: post-hook error: %s", err)
		}
	}
	logger.Infof("app: run, waiting signals...")
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

func shutdown(goctx context.Context) {
	defer logger.Infof("app: terminaled")
	for _, obj := range app.shutdown {
		ctx := NewContextV(goctx, logger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] shutdown...", obj), obj.Shutdown)
		if err != nil {
			logger.Errorf("shutdown error: %s", err)
		}
	}
}

func startup(goctx context.Context) error {
	logger.Infof("app: startup")
	for _, obj := range app.startups {
		ctx := NewContextV(goctx, logger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] startup...", obj), obj.Startup)
		if err != nil {
			return fmt.Errorf("[%T] startup error: %s", obj, err)
		}
	}
	return nil
}

func serve(goctx context.Context) error {
	logger.Infof("app: serve")
	for _, state := range app.servables {
		ctx := state.Setup(goctx)
		if statectx, ok := ctx.(*ContextV); ok && statectx.logger == nil {
			statectx.logger = logger
		}
		err := metric1(ctx, fmt.Sprintf("component[%T] start serve...", state), state.Serve)
		if err != nil {
			return fmt.Errorf("[%T] serve error: %w", state, err)
		}
	}
	return nil
}

func metric1(ctx Context, name string, step func(ctx Context) error) error {
	defer func(t time.Time) {
		ctx.Log().Infof("%s takes: %s", name, time.Since(t))
	}(time.Now())
	return step(ctx)
}

func metric2(ctx Context, name string, step func(ctx context.Context) error) error {
	return metric1(ctx, name, func(ctx Context) error {
		return step(ctx)
	})
}
