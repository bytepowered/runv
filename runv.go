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
	liveness  []Liveness
	servables []Servable
	objects   []interface{}
}

var (
	app = &application{
		initables: make([]Initable, 0, 4),
		liveness:  make([]Liveness, 0, 4),
		servables: make([]Servable, 0, 4),
		prehooks:  make([]func() error, 0, 4),
		posthooks: make([]func() error, 0, 4),
	}
	appAwaitSignal = func() <-chan os.Signal {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		return sig
	}
	appContainerd = NewContainer()
	appLogger     = NewJSONLogger()
)

func init() {
	appContainerd.AddHook(func(container *Containerd, _ interface{}) {
		container.Resolve(app)
	})
}

func SetAppLogger(l *logrus.Logger) {
	appLogger = l
}

func SetAppAwaitFunc(aaf func() <-chan os.Signal) {
	appAwaitSignal = aaf
}

// AddPrepareHook 添加PrepareHook函数
func AddPrepareHook(hook func() error) {
	app.prehooks = append(app.prehooks, hook)
}

// AddPostHook 添加PostHook函数
func AddPostHook(hook func() error) {
	app.posthooks = append(app.posthooks, hook)
}

func Provider(providerFunc interface{}) {
	Register(providerFunc)
}

func Register(in interface{}) {
	appContainerd.Register(in)
}

func Resolve(in interface{}) {
	appContainerd.Resolve(in)
}

func Container() *Containerd {
	return appContainerd
}

// Add 添加单例组件
func Add(in interface{}) {
	if in == nil {
		panic("app: add a nil object")
	}
	if act, ok := in.(Activable); ok && !act.Active() {
		return
	}
	if init, ok := in.(Initable); ok {
		app.initables = append(app.initables, init)
	}
	if live, ok := in.(Liveness); ok {
		app.liveness = append(app.liveness, live)
	}
	if servable, ok := in.(Servable); ok {
		app.servables = append(app.servables, servable)
	}
	app.objects = append(app.objects, in)
	Register(in)
}

func RunV() {
	// prepare hooks
	for _, hook := range app.prehooks {
		if err := hook(); err != nil {
			appLogger.Fatalf("app: prepare hook error: %s", err)
		}
	}
	// resolve deps
	for _, obj := range app.objects {
		Resolve(obj)
	}
	appLogger.Infof("app: init")
	// init
	for _, obj := range app.initables {
		if err := obj.OnInit(); err != nil {
			appLogger.Fatalf("app: init error: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer shutdown(goctx)
	// startup
	if err := startup(goctx); err != nil {
		appLogger.Fatalf("app: startup, error: %s", err)
	}
	// serve with setup
	if err := serve(goctx); err != nil {
		appLogger.Fatalf("app: serve, error: %s", err)
	}
	// post hook
	for _, posthook := range app.posthooks {
		if err := posthook(); err != nil {
			appLogger.Fatalf("app: post hook error: %s", err)
		}
	}
	appLogger.Infof("app: run, waiting signals...")
	<-appAwaitSignal()
}

func shutdown(goctx context.Context) {
	defer appLogger.Infof("app: terminaled")
	for _, obj := range app.liveness {
		ctx := NewStateContext(goctx, appLogger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] shutdown...", obj), obj.Shutdown)
		if err != nil {
			appLogger.Errorf("shutdown error: %s", err)
		}
	}
}

func startup(goctx context.Context) error {
	appLogger.Infof("app: startup")
	for _, obj := range app.liveness {
		ctx := NewStateContext(goctx, appLogger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] startup...", obj), obj.Startup)
		if err != nil {
			return fmt.Errorf("[%T] startup error: %s", obj, err)
		}
	}
	return nil
}

func serve(goctx context.Context) error {
	appLogger.Infof("app: serve")
	for _, state := range app.servables {
		ctx := state.Setup(goctx)
		if statectx, ok := ctx.(*StateContext); ok && statectx.logger == nil {
			statectx.logger = appLogger
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
