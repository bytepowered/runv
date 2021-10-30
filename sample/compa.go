package main

import (
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
)

var _ runv.Component = new(CompA)

type CompA struct {
}

func (c *CompA) Name() string {
	return "I am COMP A!!"
}

func (c *CompA) OnInit() error {
	logrus.Info("on init: A")
	return nil
}

func (c *CompA) Startup(ctx runv.Context) error {
	ctx.Log().Infof("startup: A")
	return nil
}

func (c *CompA) Serve(ctx runv.Context) error {
	ctx.Log().Infof("serve: A")
	return nil
}

func (c *CompA) Shutdown(ctx runv.Context) error {
	ctx.Log().Infof("shutdown: A")
	return nil
}
