package main

import (
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
)

var _ runv.Component = new(CompB)

type CompB struct {
}

func (c *CompB) SetCompA(a *CompA) {
	logrus.Info("setup: " + a.Name())
}

func (c *CompB) OnInit() error {
	logrus.Info("on init: B")

	return nil
}

func (c *CompB) Startup(ctx runv.Context) error {
	ctx.Logger().Infof("startup: B")
	return nil
}

func (c *CompB) Serve(ctx runv.Context) error {
	ctx.Logger().Infof("serve: B")
	return nil
}

func (c *CompB) Shutdown(ctx runv.Context) error {
	ctx.Logger().Infof("shutdown: B")
	return nil
}
