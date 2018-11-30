package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/k3rn3l-p4n1c/apigateway/version"
)

func noArgs(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return fmt.Errorf(
		"apigateway: '%s' is not a apigateway command.\nSee 'apigateway --help'", args[0])
}

var RootCmd = &cobra.Command{
	Use:           "apigateway [OPTIONS] COMMAND [ARG...]",
	Short:         "apigateway.",
	Long:          `apigateway.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          noArgs,
}

func init() {
	RootCmd.AddCommand(version.Cmd)
	RootCmd.AddCommand(runCmd)
}

func Execute() {
	RootCmd.Execute()
}
