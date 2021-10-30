package main

import (
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
)

func main() {
	var app runv.App = runv.NewApplication()
	app.SetInitLogger(func() *logrus.Logger {
		return logrus.New()
	})
	app.AddComponent(new(CompA))
	app.AddComponent(new(CompB))
	app.RunV()
}
