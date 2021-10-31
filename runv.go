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

var app = &wrapper{
	logger:     logrus.New(),
	initables:  make([]Initable, 0, 4),
	components: make([]Component, 0, 4),
	prepares:   make([]func() error, 0, 4),
}

type wrapper struct {
	logger     *logrus.Logger
	prepares   []func() error
	initables  []Initable
	components []Component
}

func init() {
	diRegisterProvider(logrus.New)
}

// AddPrepare 添加Prepare函数
func AddPrepare(p func() error) {
	app.prepares = append(app.prepares, p)
}

// SetLogger 通过DI注入Logger实现
func (w *wrapper) SetLogger(logger *logrus.Logger) {
	w.logger = logger
}

// AddProvider 添加Prototype对象的Provider函数
func AddProvider(providerFunc interface{}) {
	diRegisterProvider(providerFunc)
	// update app deps
	diInjectDepens(app)
}

// AddComponent 添加单例组件
func AddComponent(obj Component) {
	if init, ok := obj.(Initable); ok {
		app.initables = append(app.initables, init)
	}
	app.components = append(app.components, obj)
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
	defer doShutdown(goctx)
	if err := doStartup(goctx); err != nil {
		app.logger.Fatalf("app startup, error: %s", err)
	}
	if err := doServe(goctx); err != nil {
		app.logger.Fatalf("app serve, error: %s", err)
	}
	app.logger.Infof("app: run, waiting signals...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}

func doShutdown(goctx context.Context) {
	defer app.logger.Infof("app: terminaled")
	for _, obj := range app.components {
		ctx := newStateContext(goctx, app.logger, nil)
		err := metric(ctx, fmt.Sprintf("component[%T] shutdown...", obj), obj.Shutdown)
		if err != nil {
			app.logger.Errorf("shutdown error: %s", err)
		}
	}
}

func doStartup(goctx context.Context) error {
	app.logger.Infof("app: startup")
	for _, obj := range app.components {
		ctx := newStateContext(goctx, app.logger, nil)
		err := metric(ctx, fmt.Sprintf("component[%T] startup...", obj), obj.Startup)
		if err != nil {
			return fmt.Errorf("[%T] startup error: %s", obj, err)
		}
	}
	return nil
}

func doServe(goctx context.Context) error {
	app.logger.Infof("app: serve")
	for _, obj := range app.components {
		ctx := newStateContext(goctx, app.logger, nil)
		err := metric(ctx, fmt.Sprintf("component[%T] start serve...", obj), obj.Serve)
		if err != nil {
			return fmt.Errorf("[%T] serve error: %w", obj, err)
		}
	}
	return nil
}

func metric(ctx Context, name string, step func(ctx Context) error) error {
	defer func(t time.Time) {
		ctx.Log().Infof("%s takes: %s", name, time.Since(t))
	}(time.Now())
	return step(ctx)
}
