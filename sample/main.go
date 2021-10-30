package main

import (
	"github.com/bytepowered/runv"
	"github.com/sirupsen/logrus"
)

func main() {
	var app runv.App = runv.NewApplication()
	app.AddPrepare(func() {
		// do prepare
	})
	app.SetLogProvider(logrus.New)
	app.AddComponent(new(CompA))
	app.AddComponent(new(CompB))
	app.RunV()
}
