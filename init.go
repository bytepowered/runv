package runv

import (
	"github.com/bytepowered/runv/inject"
	"github.com/sirupsen/logrus"
	"os"
)

func init() {
	inject.RegisterProvider(NewJSONLogger)
}

func NewJSONLogger() *logrus.Logger {
	return &logrus.Logger{
		Out:          os.Stderr,
		Formatter:    new(logrus.JSONFormatter),
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.DebugLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}
}
