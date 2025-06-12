package main

import (
	"os"

	"github.com/pomdtr/tweety/internal/cmd"
)

var (
	version = "dev"
)

func main() {
	root := cmd.NewCmdRoot(version)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
