//go:build wireinject
// +build wireinject

package main

import (
	"github.com/bytepowered/runv"
	"github.com/google/wire"
	"github.com/sirupsen/logrus"
	"os"
)

func initServer() *CompServer {
	wire.Build(NewComp, NewCompServer)
	return &CompServer{}
}

func main() {
	runv.SetLogger(func() *logrus.Logger {
		return &logrus.Logger{
			Out:          os.Stderr,
			Formatter:    new(logrus.JSONFormatter),
			Hooks:        make(logrus.LevelHooks),
			Level:        logrus.DebugLevel,
			ExitFunc:     os.Exit,
			ReportCaller: false,
		}
	}())
	runv.AddPreHook(func() error {
		return nil
	})
	runv.Add(initServer())
	runv.RunV()
}
