package runv

import (
	"context"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var app = &wrapper{
	Logger:     logrus.New(),
	initables:  make([]Initable, 0, 4),
	components: make([]Component, 0, 4),
	prepares:   make([]func(), 0, 4),
}

type wrapper struct {
	Logger     *logrus.Logger
	prepares   []func()
	initables  []Initable
	components []Component
}

func AddPrepare(p func()) {
	app.prepares = append(app.prepares, p)
}

// SetLogger 通过DI注入Logger实现
func (a *wrapper) SetLogger(logger *logrus.Logger) {
	a.Logger = logger
}

func init() {
	diRegisterProvider(logrus.New)
}

func RegisterComponentProvider(providerFunc interface{}) {
	diRegisterProvider(providerFunc)
	// update app deps
	diInjectDepens(app)
}

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
		pre()
	}
	app.Logger.Infof("app: init")
	// init and inject deps
	for _, obj := range app.initables {
		diInjectDepens(obj)
		if err := obj.OnInit(); err != nil {
			app.Logger.Fatalf("init failed: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer func() {
		for _, cmp := range app.components {
			ctx := newStateContext(goctx, app.Logger, nil)
			err := metric(ctx, "comp shutdown", cmp.Shutdown)
			if err != nil {
				app.Logger.Errorf("shutdown error: %s", err)
			}
		}
		app.Logger.Infof("app: terminaled")
	}()
	// start
	app.Logger.Infof("app: start")
	for _, cmp := range app.components {
		ctx := newStateContext(goctx, app.Logger, nil)
		err := metric(ctx, "comp startup", cmp.Startup)
		if err != nil {
			app.Logger.Errorf("startup error: %s", err)
		}
	}
	// serve
	app.Logger.Infof("app: serve")
	for _, cmp := range app.components {
		ctx := newStateContext(goctx, app.Logger, nil)
		err := metric(ctx, "comp serve", cmp.Serve)
		if err != nil {
			app.Logger.Errorf("serve error: %s", err)
		}
	}
	app.Logger.Infof("app: run, waiting signals...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}

func metric(ctx Context, name string, step func(ctx Context) error) error {
	defer func(t time.Time) {
		ctx.Log().Infof("%s takes: %s", name, time.Since(t))
	}(time.Now())
	return step(ctx)
}
