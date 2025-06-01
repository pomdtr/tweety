package main

import (
	"os"

	"github.com/pomdtr/tweety/internal/cmd"
)

func main() {
	root := cmd.NewCmdRoot()

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
