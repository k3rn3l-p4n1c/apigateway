package main

import (
	"fmt"
	"github.com/k3rn3l-p4n1c/goaway/ctl/cmd"
	"os"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
