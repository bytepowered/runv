package runv

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Context interface {
	context.Context
	ValueE(key interface{}) (value interface{}, ok bool)
	Log() *logrus.Logger
}

type Initable interface {
	OnInit() error
}

type StateComponent interface {
	Setup(ctx context.Context) Context
	Serve(Context) error
}

type Component interface {
	Startup(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
