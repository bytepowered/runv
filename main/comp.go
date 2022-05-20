package main

import (
	"context"
	"fmt"
	"github.com/bytepowered/runv"
)

var _ runv.Liveness = new(Comp)

type Comp struct {
}

func NewComp() *Comp {
	return new(Comp)
}

func (c *Comp) Startup(ctx context.Context) error {
	fmt.Println("startup: B")
	return nil
}

func (c *Comp) Shutdown(ctx context.Context) error {
	fmt.Println("shutdown: B")
	return nil
}
