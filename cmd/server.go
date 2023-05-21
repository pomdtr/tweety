package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/joho/godotenv"
	"github.com/pomdtr/webterm/server"
	"github.com/spf13/cobra"
)

func NewCmdServer() *cobra.Command {
	return &cobra.Command{
		Use:    "server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			envFile := filepath.Join(xdg.ConfigHome, "webterm", "webterm.env")
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
			server := server.NewServer(messageHandler, environ)

			go messageHandler.Loop()
			if err := server.ListenAndServe(); err != nil {
				return err
			}

			return nil
		},
	}
}
