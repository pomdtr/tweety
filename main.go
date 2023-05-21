package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/pomdtr/webterm/cmd"
)

func main() {
	logFile := os.Getenv("WEBTERM_LOGFILE")
	if logFile == "" {
		logFile = filepath.Join(xdg.StateHome, "webterm", "log")
	}

	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		fmt.Printf("unable to create log directory: %v\n", err)
		os.Exit(1)
	}

	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("unable to open log file: %v\n", err)
		os.Exit(1)
	}

	log.Default().SetOutput(f)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
