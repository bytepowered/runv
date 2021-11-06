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
	// OnInit 初始化组件
	// 此方法执行时，如果返回非nil的error，整个服务启动过程将被终止。
	OnInit() error
}

type StateComponent interface {
	// Setup 创建并返回Context；后续Serve方法调用时，将使用此Context作为参数。
	Setup(ctx context.Context) Context

	// Serve 基于Context执行服务；
	// 此方法执行时，如果返回非nil的error，整个服务启动过程将被终止。
	Serve(Context) error
}

type Component interface {
	// Startup 用于启动组件的生命周期方法；
	// 此方法执行时，如果返回非nil的error，整个服务启动过程将被终止。
	Startup(ctx context.Context) error

	// Shutdown 用于停止组件的生命周期方法；
	// 如果返回非nil的error，将打印日志记录；
	Shutdown(ctx context.Context) error
}
