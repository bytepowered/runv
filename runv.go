package runv

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	app = &wrapper{
		logger:     logrus.New(),
		initables:  make([]Initable, 0, 4),
		components: make([]Component, 0, 4),
		states:     make([]StateComponent, 0, 4),
		prepares:   make([]func() error, 0, 4),
	}
	appAwait = func() <-chan os.Signal {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		return sig
	}
)

type wrapper struct {
	logger     *logrus.Logger
	prepares   []func() error
	initables  []Initable
	components []Component
	states     []StateComponent
}

// SetLogger 通过DI注入Logger实现
func (w *wrapper) SetLogger(logger *logrus.Logger) {
	w.logger = logger
}

func init() {
	diRegisterProvider(logrus.New)
}

func SetAppAwaitFunc(saf func() <-chan os.Signal) {
	appAwait = saf
}

// AddPrepareHook 添加Prepare函数
func AddPrepareHook(p func() error) {
	app.prepares = append(app.prepares, p)
}

// Provider 添加Prototype对象的Provider函数
func Provider(providerFunc interface{}) {
	diRegisterProvider(providerFunc)
	// update app deps
	diInjectDepens(app)
}

// Add 添加单例组件
func Add(obj interface{}) {
	if init, ok := obj.(Initable); ok {
		app.initables = append(app.initables, init)
	}
	if comp, ok := obj.(Component); ok {
		app.components = append(app.components, comp)
	}
	if state, ok := obj.(StateComponent); ok {
		app.states = append(app.states, state)
	}
	diRegisterObject(obj)
	// update app deps
	diInjectDepens(app)
}

func RunV() {
	// prepare
	for _, pre := range app.prepares {
		if err := pre(); err != nil {
			app.logger.Fatalf("app: prepare, %s", err)
		}
	}
	app.logger.Infof("app: init")
	// init and inject deps
	for _, obj := range app.initables {
		diInjectDepens(obj)
		if err := obj.OnInit(); err != nil {
			app.logger.Fatalf("init failed: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer shutdown(goctx)
	if err := startup(goctx); err != nil {
		app.logger.Fatalf("app startup, error: %s", err)
	}
	if err := serve(goctx); err != nil {
		app.logger.Fatalf("app serve, error: %s", err)
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
