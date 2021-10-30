package runv

import (
	"context"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var _ App = new(Application)

type AppOptions func(*Application)

type Application struct {
	Logger     *logrus.Logger
	fLogger    func() *logrus.Logger
	prepares   []func()
	initables  []Initable
	components []Component
}

func (a *Application) AddPrepare(p func()) {
	a.prepares = append(a.prepares, p)
}

func (a *Application) SetLogProvider(f func() *logrus.Logger) {
	a.fLogger = f
}

func NewApplication() *Application {
	return &Application{
		Logger:     logrus.New(),
		initables:  make([]Initable, 0, 4),
		components: make([]Component, 0, 4),
		prepares:   make([]func(), 0, 4),
	}
}

func (a *Application) RegisterComponentProvider(providerFunc interface{}) {
	diRegisterProvider(providerFunc)
}

func (a *Application) AddComponent(obj Component) {
	if init, ok := obj.(Initable); ok {
		a.initables = append(a.initables, init)
	}
	a.components = append(a.components, obj)
	diRegisterObject(obj)
}

func (a *Application) RunV() {
	if a.fLogger != nil {
		a.Logger = a.fLogger()
	}
	// prepare
	for _, pre := range a.prepares {
		pre()
	}
	a.Logger.Infof("app: init")
	// init and inject deps
	for _, obj := range a.initables {
		diInjectDepens(obj)
		if err := obj.OnInit(); err != nil {
			a.Logger.Fatalf("init failed: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
	// finally shutdown
	defer func() {
		for _, cmp := range a.components {
			ctx := newStateContext(goctx, a.Logger, nil)
			err := a.metric(ctx, "comp shutdown", cmp.Shutdown)
			if err != nil {
				a.Logger.Errorf("shutdown error: %s", err)
			}
		}
		a.Logger.Infof("app: terminaled")
	}()
	// start
	a.Logger.Infof("app: start")
	for _, cmp := range a.components {
		ctx := newStateContext(goctx, a.Logger, nil)
		err := a.metric(ctx, "comp startup", cmp.Startup)
		if err != nil {
			a.Logger.Errorf("startup error: %s", err)
		}
	}
	// serve
	a.Logger.Infof("app: serve")
	for _, cmp := range a.components {
		ctx := newStateContext(goctx, a.Logger, nil)
		err := a.metric(ctx, "comp serve", cmp.Serve)
		if err != nil {
			a.Logger.Errorf("serve error: %s", err)
		}
	}
	a.Logger.Infof("app: run, waiting signals...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
}

func (a *Application) metric(ctx Context, name string, step func(ctx Context) error) error {
	defer func(t time.Time) {
		a.Logger.Infof("%s takes: %s", name, time.Since(t))
	}(time.Now())
	return step(ctx)
}
