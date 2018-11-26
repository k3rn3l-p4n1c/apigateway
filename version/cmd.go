package version

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Check version",
	Run: func(cmd *cobra.Command, args []string) {
		PrintVersion()
	},
}
