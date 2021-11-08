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
	w = &wrapper{
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

func SetAppAwaitFunc(saf func() <-chan os.Signal) {
	appAwait = saf
}

// AddPrepareHook 添加PrepareHook函数
func AddPrepareHook(hook func() error) {
	w.prehooks = append(w.prehooks, hook)
}

// AddPostHook 添加PostHook函数
func AddPostHook(hook func() error) {
	w.posthooks = append(w.posthooks, hook)
}

// Provider 添加Prototype对象的Provider函数
func Provider(providerFunc interface{}) {
	inject.RegisterProvider(providerFunc)
	// update app deps
	inject.ResolveDeps(w)
}

// Add 添加单例组件
func Add(obj interface{}) {
	if obj == nil {
		panic("app: add a nil object")
	}
	hits := 0
	if init, ok := obj.(Initable); ok {
		hits++
		w.initables = append(w.initables, init)
	}
	if comp, ok := obj.(Component); ok {
		hits++
		w.components = append(w.components, comp)
	}
	if state, ok := obj.(StateComponent); ok {
		hits++
		w.states = append(w.states, state)
	}
	if hits == 0 {
		panic(fmt.Errorf("app: add an unsupported component, type: %T ", obj))
	}
	w.refs = append(w.refs, obj)
	inject.RegisterObject(obj)
	// update app deps
	inject.ResolveDeps(w)
}

func RunV() {
	// prepare hooks
	for _, prehook := range w.prehooks {
		if err := prehook(); err != nil {
			w.logger.Fatalf("app: prepare hook error: %s", err)
		}
	}
	// inject deps
	for _, obj := range w.refs {
		inject.ResolveDeps(obj)
	}
	w.logger.Infof("app: init")
	// init
	for _, obj := range w.initables {
		if err := obj.OnInit(); err != nil {
			w.logger.Fatalf("app: init error: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer shutdown(goctx)
	// startup
	if err := startup(goctx); err != nil {
		w.logger.Fatalf("app startup, error: %s", err)
	}
	// serve with setup
	if err := serve(goctx); err != nil {
		w.logger.Fatalf("app serve, error: %s", err)
	}
	// post hook
	for _, posthook := range w.posthooks {
		if err := posthook(); err != nil {
			w.logger.Fatalf("app: post hook error: %s", err)
		}
	}
	w.logger.Infof("app: run, waiting signals...")
	<-appAwait()
}

func shutdown(goctx context.Context) {
	defer w.logger.Infof("app: terminaled")
	for _, obj := range w.components {
		ctx := NewStateContext(goctx, w.logger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] shutdown...", obj), obj.Shutdown)
		if err != nil {
			w.logger.Errorf("shutdown error: %s", err)
		}
	}
}

func startup(goctx context.Context) error {
	w.logger.Infof("app: startup")
	for _, obj := range w.components {
		ctx := NewStateContext(goctx, w.logger, nil)
		err := metric2(ctx, fmt.Sprintf("component[%T] startup...", obj), obj.Startup)
		if err != nil {
			return fmt.Errorf("[%T] startup error: %s", obj, err)
		}
	}
	return nil
}

func serve(goctx context.Context) error {
	w.logger.Infof("app: serve")
	for _, state := range w.states {
		ctx := state.Setup(goctx)
		if statectx, ok := ctx.(*StateContext); ok && statectx.logger == nil {
			statectx.logger = w.logger
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
