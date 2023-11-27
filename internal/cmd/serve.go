package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/phayes/freeport"
	"github.com/pomdtr/popcorn/internal/server"
	"github.com/sethvargo/go-password/password"
	"github.com/spf13/cobra"
)

var runtimeDir = filepath.Join(xdg.RuntimeDir, "popcorn")

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "serve",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(runtimeDir); err != nil && os.IsNotExist(err) {
				if err := os.MkdirAll(runtimeDir, 0755); err != nil {
					return fmt.Errorf("could not create runtime dir: %w", err)
				}

				return fmt.Errorf("could not check for runtime dir presence: %w", err)
			}

			port, err := freeport.GetFreePort()
			if err != nil {
				return fmt.Errorf("could not find freeport %w", err)
			}

			token, err := password.Generate(32, 10, 0, false, true)
			if err != nil {
				return fmt.Errorf("could not generate secret %w", err)
			}

			messageHandler := server.NewMessageHandler()
			go messageHandler.Loop()
			messageHandler.SendMessage(map[string]any{
				"command": "init",
				"port":    port,
				"token":   token,
			})

			if err := server.Serve(messageHandler, port, token); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
