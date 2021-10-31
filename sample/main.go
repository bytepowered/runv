package main

import (
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
	"os"
)

func newJSONLogger() *logrus.Logger {
	return &logrus.Logger{
		Out:          os.Stderr,
		Formatter:    new(logrus.JSONFormatter),
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
}

func main() {
	runv.AddProvider(newJSONLogger)
	runv.AddPrepare(func() error {
		// do prepare
		return nil
	})
	runv.AddComponent(new(CompA))
	runv.AddComponent(new(CompB))
	runv.RunV()
}
