package main

import (
	"context"
	"fmt"
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
)

var _ runv.Component = new(CompB)

type CompB struct {
}

func (c *CompB) SetCompA(a *CompA) {
	logrus.Info("di set: " + a.Name())
}

func (c *CompB) InjectCompA(a *CompA) {
	logrus.Info("di inject: " + a.Name())
}

func (c *CompB) Startup(ctx context.Context) error {
	fmt.Println("startup: B")
	return nil
}

func (c *CompB) Shutdown(ctx context.Context) error {
	fmt.Println("shutdown: B")
	return nil
}
