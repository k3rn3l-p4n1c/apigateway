package cmd

import (
	"github.com/spf13/cobra"
	"github.com/k3rn3l-p4n1c/apigateway/engine"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run server",
	Run:   Run,
}

func Run(cmd *cobra.Command, args []string) {
	e := engine.Engine{}
	e.Start()
}

func init() {
}
