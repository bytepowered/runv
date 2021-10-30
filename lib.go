package runv

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Context interface {
	Context() context.Context
	GetVarE(key interface{}) (value interface{}, ok bool)
	GetVar(key interface{}) (value interface{})
	Logger() *logrus.Logger
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
	SetInitLogger(func() *logrus.Logger)
	AddComponent(component Component)
	RunV()
}
