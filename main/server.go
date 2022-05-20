package main

import (
	"context"
	"fmt"
	"github.com/bytepowered/runv"
)

var (
	_ runv.Liveness = new(CompServer)
	_ runv.Servable = new(CompServer)
)

type CompServer struct {
	comp *Comp
}

func NewCompServer(cmp *Comp) *CompServer {
	return &CompServer{
		comp: cmp,
	}
}

func (c *CompServer) Name() string {
	return "I am COMP Server!!"
}

func (c *CompServer) Startup(ctx context.Context) error {
	fmt.Println("Server startup")
	return nil
}

func (c *CompServer) Serve(ctx context.Context) error {
	runv.Log().Infof("Server serve")
	return nil
}

func (c *CompServer) Shutdown(ctx context.Context) error {
	fmt.Println("Server shutdown")
	return nil
}
