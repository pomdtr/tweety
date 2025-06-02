package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "tweety",
		SilenceUsage:      true,
		Short:             "An integrated terminal for your web browser",
		ValidArgsFunction: completeCommand,
		Args:              cobra.ArbitraryArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			confmapProvider := confmap.Provider(map[string]interface{}{
				"command": getDefaultShell(),
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Usage()
			}

			// First try to find the exact file name
			entrypoint := filepath.Join(commandDir, args[0])
			stat, err := os.Stat(entrypoint)

			// If not found, try to find any file that starts with the command name
			if os.IsNotExist(err) {
				files, readErr := os.ReadDir(commandDir)
				if readErr == nil {
					for _, file := range files {
						if file.IsDir() {
							continue
						}

						name := file.Name()
						nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))

						if nameWithoutExt == args[0] {
							entrypoint = filepath.Join(commandDir, name)
							stat, err = os.Stat(entrypoint)
							break
						}
					}
				}
			}

			if err != nil {
				return fmt.Errorf("unknown command: %s", args[0])
			}

			if stat.IsDir() {
				return fmt.Errorf("command entrypoint is a directory, expected a file: %s", entrypoint)
			}

			// check if the entrypoint is executable
			if stat.Mode()&0111 == 0 {
				if err := os.Chmod(entrypoint, 0755); err != nil {
					return fmt.Errorf("failed to make command entrypoint executable: %w", err)
				}
			}

			cmdExec := exec.Command(entrypoint, args[1:]...)

			cmdExec.Stdin = os.Stdin
			cmdExec.Stdout = os.Stdout
			cmdExec.Stderr = os.Stderr

			cmd.SilenceErrors = true
			return cmdExec.Run()
		},
	}

	cmd.Flags().SetInterspersed(true)

	cmd.AddCommand(
		NewCmdServe(),
		NewCmdInstall(),
		NewCmdUninstall(),
		NewCmdTabs(),
		NewCmdBookmarks(),
		NewCmdHistory(),
		NewCmdWindows(),
		NewCmdNotifications(),
		NewCmdEdit(),
		NewCmdSSH(),
		NewCmdOpen(),
		NewCmdConfig(),
		NewCmdApps(),
	)

	return cmd
}

func completeCommand(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	files, err := os.ReadDir(commandDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var commands []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		// Strip any extension for command completion
		name = strings.TrimSuffix(name, filepath.Ext(name))
		commands = append(commands, name)
	}

	return commands, cobra.ShellCompDirectiveNoFileComp
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
