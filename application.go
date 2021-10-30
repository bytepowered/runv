package runv

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"reflect"
	"time"
)

var _ App = new(Application)

type AppOptions func(*Application)

type Application struct {
	Logger     *logrus.Logger
	fLogger    func() *logrus.Logger
	prepare    []func()
	initables  []Initable
	components []Component
	typerefs   map[string]reflect.Value
}

func (a *Application) AddPrepare(p func()) {
	a.prepare = append(a.prepare, p)
}

func (a *Application) SetInitLogger(f func() *logrus.Logger) {
	a.fLogger = f
}

func NewApplication() *Application {
	return &Application{
		Logger:     logrus.New(),
		initables:  make([]Initable, 0, 4),
		components: make([]Component, 0, 4),
		typerefs:   make(map[string]reflect.Value, 4),
		prepare:    make([]func(), 0, 4),
	}
}

func (a *Application) AddComponent(cmp Component) {
	if init, ok := cmp.(Initable); ok {
		a.initables = append(a.initables, init)
	}
	a.components = append(a.components, cmp)
	tn := fmt.Sprintf("%T", cmp)
	fmt.Printf("add comp, type: %s\n", tn)
	a.typerefs[tn] = reflect.ValueOf(cmp)
}

func (a *Application) RunV() {
	if a.fLogger != nil {
		a.Logger = a.fLogger()
	}
	a.Logger.Infof("app: init")
	// init
	injectloader := func(typestr string) ([]reflect.Value, bool) {
		v, ok := a.typerefs[typestr]
		return []reflect.Value{v}, ok
	}
	for _, init := range a.initables {
		inject(init, injectloader)
		err := init.OnInit()
		if err != nil {
			a.Logger.Fatalf("init failed: %s", err)
		}
	}
	goctx, ctxfun := context.WithCancel(context.Background())
	defer ctxfun()
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
	a.Logger.Infof("app: started")
	// serve
	a.Logger.Infof("app: serve")
	for _, cmp := range a.components {
		ctx := newStateContext(goctx, a.Logger, nil)
		err := a.metric(ctx, "comp serve", cmp.Serve)
		if err != nil {
			a.Logger.Errorf("serve error: %s", err)
		}
	}
	a.Logger.Infof("app: runed")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func (a *Application) metric(ctx Context, name string, step func(ctx Context) error) error {
	defer func(t time.Time) {
		a.Logger.Infof("%s takes: %s", name, time.Since(t))
	}(time.Now())
	return step(ctx)
}
