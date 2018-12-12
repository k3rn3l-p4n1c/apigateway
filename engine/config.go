package engine

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/k3rn3l-p4n1c/apigateway"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	configFilePath = "./config.yml"
)

func Load() (*apigateway.Config, error) {
	v := viper.New()

	v.SetConfigFile(configFilePath)

	v.OnConfigChange(OnConfigChanged)
	v.WatchConfig()

	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("can't read v file error=(%v)", err)
	}
	v.SetDefault("log_level", "debug")
	setLogLevel(v.GetString("log_level"))
	logrus.Debug(v.AllSettings())
	var config = apigateway.Config{}

	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

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

// OnConfigChanged excuates when config changes
func OnConfigChanged(_ fsnotify.Event) {
	logrus.Info("configuration is reloaded")
	// todo
}
