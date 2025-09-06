package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	jsonparser "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"

	"github.com/knadh/koanf/v2"

	"github.com/spf13/cobra"
)

var k = koanf.New(".")

var (
	maxBufferSizeBytes   = 512
	keepalivePingTimeout = 20 * time.Second
)

var configDir = filepath.Join(os.Getenv("HOME"), ".config", "tweety")
var cacheDir = filepath.Join(os.Getenv("HOME"), ".cache", "tweety")
var dataDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "tweety")
var commandDir = filepath.Join(configDir, "commands")
var appDir = filepath.Join(configDir, "apps")

func NewCmdRoot(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "tweety",
		SilenceUsage: true,
		Short:        "An integrated terminal for your web browser",
		Version:      version,
		Args:         cobra.ArbitraryArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			confmapProvider := confmap.Provider(map[string]interface{}{
				"command": getDefaultShell(),
				"theme":   "Tomorrow Night",
			}, ".")
			if err := k.Load(confmapProvider, nil); err != nil {
				return fmt.Errorf("failed to load default config: %w", err)
			}

			configPath := filepath.Join(configDir, "config.json")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				if err := os.MkdirAll(configDir, 0755); err != nil {
					return fmt.Errorf("failed to create config directory: %w", err)
				}

				configBytes, err := json.MarshalIndent(map[string]interface{}{
					"command": getDefaultShell(),
					"editor":  getDefaultEditor(),
				}, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal default config: %w", err)
				}

				if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
					return fmt.Errorf("failed to write default config: %w", err)
				}
			}

			f := file.Provider(filepath.Join(configDir, "config.json"))
			if err := k.Load(f, jsonparser.Parser()); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			f.Watch(func(event interface{}, err error) {
				if err != nil {
					log.Printf("watch error: %v", err)
					return
				}

				k = koanf.New(".")
				k.Load(f, jsonparser.Parser())
			})

			return nil
		},
	}

	cmd.Flags().SetInterspersed(true)

	cmd.AddCommand(
		NewCmdServe(),
		NewCmdInstall(),
		NewCmdTabs(),
		NewCmdBookmarks(),
		NewCmdEdit(),
		NewCmdHistory(),
		NewCmdWindows(),
		NewCmdNotifications(),
		NewCmdRun(),
		NewCmdOpen(),
		NewCmdFetch(),
	)

	return cmd
}

func getDefaultShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		switch runtime.GOOS {
		case "darwin":
			return "/bin/zsh"
		case "linux":
			return "/bin/bash"
		default:
			return "/bin/sh"
		}
	}
	return shell
}

func getDefaultEditor() string {
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}

	return "vi"
}
