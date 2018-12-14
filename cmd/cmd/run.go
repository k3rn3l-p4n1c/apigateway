package cmd

import (
	"github.com/k3rn3l-p4n1c/apigateway/engine"
	"github.com/spf13/cobra"
	"github.com/sirupsen/logrus"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run server",
	Run:   Run,
}

func Run(cmd *cobra.Command, args []string) {
	e, err := engine.NewEngine()
	if err != nil {
		logrus.WithError(err).Fatal("unable to load engine.")
	}
	e.Start()
}

func init() {
}
