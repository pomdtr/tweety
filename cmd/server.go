package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/pomdtr/wsh/server"
	"github.com/spf13/cobra"
)

const wshPort = 9999

func NewCmdServer() *cobra.Command {
	return &cobra.Command{
		Use:    "server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			configDir := filepath.Join(homeDir, ".config", "wsh")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return err
			}

			envFile := filepath.Join(configDir, "wsh.env")
			environ := os.Environ()
			environ = append(environ, "TERM=xterm-256color")

			if _, err := os.Stat(envFile); err == nil {
				env, err := godotenv.Read(envFile)
				if err != nil {
					return err
				}

				for k, v := range env {
					environ = append(environ, fmt.Sprintf("%s=%s", k, v))
				}
			}

			messageHandler := server.NewMessageHandler()
			go messageHandler.Loop()

			if err := server.Serve(messageHandler, wshPort, environ); err != nil {
				return err
			}

			return nil
		},
	}
}
