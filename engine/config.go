package engine

import (
	"github.com/sirupsen/logrus"
)

var (
	configFilePath = "./config.yml"
)


// SetDebugLogLevel sets log level to debug mode
func setLogLevel(logLevel string) {
	switch logLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	default:
		logrus.WithField("log_level", logLevel).Fatal("invalid value for field log_level")
	}
}
