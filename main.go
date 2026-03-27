package main

import (
	"os"

	"github.com/oobagi/notebook/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		cmd.PrintErrorStderr(err.Error())
		os.Exit(1)
	}
}
