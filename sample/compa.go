package main

import (
	"context"
	"fmt"
	"github.com/bytepowered/runv"
)

var (
	_ runv.Liveness = new(CompA)
	_ runv.Servable = new(CompA)
)

type CompA struct {
}

func (c *CompA) Name() string {
	return "I am COMP A!!"
}

func (c *CompA) Startup(ctx context.Context) error {
	fmt.Println("startup: A")
	return nil
}

func (c *CompA) Setup(ctx context.Context) runv.Context {
	return runv.NewContextV0(ctx)
}

func (c *CompA) Serve(ctx runv.Context) error {
	runv.Log().Infof("serve: A")
	return nil
}

func (c *CompA) Shutdown(ctx context.Context) error {
	fmt.Println("shutdown: A")
	return nil
}
