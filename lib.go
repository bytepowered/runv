package runv

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Context interface {
	Context() context.Context
	GetVarE(key interface{}) (value interface{}, ok bool)
	GetVar(key interface{}) (value interface{})
	Log() *logrus.Logger
}

type Initable interface {
	OnInit() error
}

type Component interface {
	Startup(Context) error
	Serve(Context) error
	Shutdown(Context) error
}

type App interface {
	AddPrepare(func())
	RegisterComponentProvider(providerFunc interface{})
	AddComponent(componentObj Component)
	RunV()
}
