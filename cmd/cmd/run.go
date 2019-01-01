package cmd

import (
	"github.com/k3rn3l-p4n1c/apigateway/engine"
	"github.com/spf13/cobra"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run server",
	Run:   Run,
}

func Run(cmd *cobra.Command, args []string) {
	v := viper.New()

	v.SetConfigFile(configFilePath)
	err := v.ReadInConfig()
	if err != nil {
		logrus.Fatal("can't read v file error=(%v)", err)
	}
	v.SetDefault("log_level", "debug")

	setLogLevel(v.GetString("log_level"))
	e, err := engine.NewEngine(v)
	if err != nil {
		logrus.WithError(err).Fatal("unable to load engine.")
	}

	v.OnConfigChange(e.OnConfigChange)
	v.WatchConfig()

	e.Start()
}

func init() {
}
