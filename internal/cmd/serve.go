package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/fsnotify/fsnotify"
	"github.com/phayes/freeport"
	"github.com/pomdtr/popcorn/internal/config"
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

			cfg, err := config.Load(config.Path)
			if err != nil {
				return fmt.Errorf("could not load config %w", err)
			}

			messageHandler := server.NewMessageHandler()
			go messageHandler.Loop()
			messageHandler.SendMessage(map[string]any{
				"command": "init",
				"port":    port,
				"config":  cfg,
				"token":   token,
			})

			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				return fmt.Errorf("could not create watcher %w", err)
			}
			defer watcher.Close()

			go func() {
				for {
					select {
					case event, ok := <-watcher.Events:
						if !ok {
							return
						}
						if event.Has(fsnotify.Write) {
							cfg, err := config.Load(config.Path)
							if err != nil {
								log.Println("could not load config", err)
								continue
							}

							messageHandler.SendMessage(map[string]any{
								"command": "config",
								"config":  cfg,
							})
						}
					case err, ok := <-watcher.Errors:
						if !ok {
							return
						}
						log.Println("error:", err)
					}
				}
			}()

			if err := watcher.Add(config.Path); err != nil {
				return fmt.Errorf("could not watch config file %w", err)
			}

			if err := server.Serve(messageHandler, port, token); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
