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
	"github.com/bytepowered/runv/inject"
	"github.com/sirupsen/logrus"
)

var (
	app = &wrapper{
		logger:     logrus.New(),
		initables:  make([]Initable, 0, 4),
		components: make([]Component, 0, 4),
		states:     make([]StateComponent, 0, 4),
		prehooks:   make([]func() error, 0, 4),
		posthooks:  make([]func() error, 0, 4),
	}
	appAwait = func() <-chan os.Signal {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		return sig
	}
)

type wrapper struct {
	logger     *logrus.Logger
	prehooks   []func() error
	posthooks  []func() error
	initables  []Initable
	components []Component
	states     []StateComponent
	refs       []interface{}
}

// SetLogger 通过DI注入Logger实现
func (w *wrapper) SetLogger(logger *logrus.Logger) {
	w.logger = logger
}

func init() {
	inject.RegisterProvider(&logrus.Logger{
		Out:          os.Stderr,
		Formatter:    new(logrus.JSONFormatter),
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	})
}

func SetAppAwaitFunc(saf func() <-chan os.Signal) {
	appAwait = saf
}

// AddPrepareHook 添加PrepareHook函数
func AddPrepareHook(hook func() error) {
	app.prehooks = append(app.prehooks, hook)
}

// AddPostHook 添加PostHook函数
func AddPostHook(hook func() error) {
	app.posthooks = append(app.posthooks, hook)
}

// Provider 添加Prototype对象的Provider函数
func Provider(providerFunc interface{}) {
	inject.RegisterProvider(providerFunc)
	// update app deps
	inject.ResolveDeps(app)
}

// Add 添加单例组件
func Add(obj interface{}) {
	if obj == nil {
		panic("app: add a nil object")
	}
	hits := 0
	if init, ok := obj.(Initable); ok {
		hits++
		app.initables = append(app.initables, init)
	}
	if comp, ok := obj.(Component); ok {
		hits++
		app.components = append(app.components, comp)
	}
	if state, ok := obj.(StateComponent); ok {
		hits++
		app.states = append(app.states, state)
	}
	if hits == 0 {
		panic(fmt.Errorf("app: add an unsupported component, type: %T ", obj))
	}
	app.refs = append(app.refs, obj)
	inject.RegisterObject(obj)
	// update app deps
	inject.ResolveDeps(app)
}

func RunV() {
	// prepare hooks
	for _, prehook := range app.prehooks {
		if err := prehook(); err != nil {
			app.logger.Fatalf("app: prepare hook error: %s", err)
		}
	}
	// inject deps
	for _, obj := range app.refs {
		inject.ResolveDeps(obj)
	}
	app.logger.Infof("app: init")
	// init
	for _, obj := range app.initables {
		if err := obj.OnInit(); err != nil {
			app.logger.Fatalf("app: init error: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer shutdown(goctx)
	// startup
	if err := startup(goctx); err != nil {
		app.logger.Fatalf("app startup, error: %s", err)
	}
	// serve with setup
	if err := serve(goctx); err != nil {
		app.logger.Fatalf("app serve, error: %s", err)
	}
	// post hook
	for _, posthook := range app.posthooks {
		if err := posthook(); err != nil {
			app.logger.Fatalf("app: post hook error: %s", err)
		}
	}
	app.logger.Infof("app: run, waiting signals...")
	<-appAwait()
}

func shutdown(goctx context.Context) {
	defer app.logger.Infof("app: terminaled")
	for _, obj := range app.components {
		ctx := NewStateContext(goctx, app.logger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] shutdown...", obj), obj.Shutdown)
		if err != nil {
			app.logger.Errorf("shutdown error: %s", err)
		}
	}
}

func startup(goctx context.Context) error {
	app.logger.Infof("app: startup")
	for _, obj := range app.components {
		ctx := NewStateContext(goctx, app.logger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] startup...", obj), obj.Startup)
		if err != nil {
			return fmt.Errorf("[%T] startup error: %s", obj, err)
		}
	}
	return nil
}

func serve(goctx context.Context) error {
	app.logger.Infof("app: serve")
	for _, state := range app.states {
		ctx := state.Setup(goctx)
		if statectx, ok := ctx.(*StateContext); ok && statectx.logger == nil {
			statectx.logger = app.logger
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
