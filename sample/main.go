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
	var app runv.App = runv.NewApplication()
	app.AddPrepare(func() {
		// do prepare
	})
	app.RegisterComponentProvider(newJSONLogger)
	app.AddComponent(new(CompA))
	app.AddComponent(new(CompB))
	app.RunV()
}
