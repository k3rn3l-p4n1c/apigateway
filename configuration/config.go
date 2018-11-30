package configuration

import (
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"errors"
)

var (
	configFilePath = "./config.yml"
)


func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configFilePath)

	v.OnConfigChange(OnConfigChanged)
	v.WatchConfig()

	err := v.ReadInConfig()
	if err != nil {
		return nil, errors.New("can't read v file")
	}
	SetDebugLogLevel(true)
	var config = Config{}

	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	name2service := make(map[string]*Backend)
	for _, backend := range config.Backend {
		name2service[backend.Name] = backend
	}
	for _, frontend := range config.Frontend {
		frontend.ToBackend = name2service[frontend.To]
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
