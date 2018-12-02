package engine

import (
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"errors"
	"github.com/k3rn3l-p4n1c/apigateway"
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
		return nil, errors.New("can't read v file")
	}
	v.SetDefault("entryPoints.enabled", true)
	SetDebugLogLevel(true)
	logrus.Debug(v.AllSettings())
	var config = apigateway.Config{}

	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SetDebugLogLevel sets log level to debug mode
func SetDebugLogLevel(isDebug bool) {
	if isDebug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("log level is set to Debug")
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
}

// OnConfigChanged excuates when config changes
func OnConfigChanged(_ fsnotify.Event) {
	logrus.Info("configuration is reloaded")
	// todo
}
