package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pomdtr/tweety/extension"
	"github.com/spf13/cobra"
)

//go:embed all:embed
var embedFs embed.FS

func NewCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Tweety extension",
		Long:  "Installs the Tweety browser extension and sets up the native messaging host.",
	}

	cmd.AddCommand(NewCmdInstallExtension())
	cmd.AddCommand(NewCmdInstallManifest())

	return cmd
}

func NewCmdInstallExtension() *cobra.Command {
	var flags struct {
		overwrite bool
	}

	cmd := &cobra.Command{
		Use:   "extension <dir>",
		Short: "Install extension",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			extensionDir := filepath.Join(args[0])
			if err := os.MkdirAll(extensionDir, 0755); err != nil {
				return fmt.Errorf("failed to create extension directory: %w", err)
			}

			if _, err := os.Stat(extensionDir); err == nil {
				if !flags.overwrite {
					return fmt.Errorf("extension already installed, use --overwrite to reinstall")
				}

				if err := os.RemoveAll(extensionDir); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove existing extension directory: %w", err)
				}
			}

			if err := os.CopyFS(extensionDir, extension.FS); err != nil {
				return fmt.Errorf("failed to copy extension files: %w", err)
			}

			cmd.Printf("Extension installed successfully at %s\n", extensionDir)
			return nil

		},
	}

	cmd.Flags().BoolVar(&flags.overwrite, "overwrite", false, "Overwrite existing native messaging host and manifest files")

	return cmd
}

func NewCmdInstallManifest() *cobra.Command {
	return &cobra.Command{
		Use:   "manifest",
		Short: "Install native messaging host manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %w", err)
			}

			hostTemplate, err := template.ParseFS(embedFs, "embed/native_messaging_host.tmpl")
			if err != nil {
				return fmt.Errorf("failed to parse template: %w", err)
			}

			hostPath := filepath.Join(dataDir, "native_messaging_host")
			f, err := os.Create(hostPath)
			if err != nil {
				return fmt.Errorf("failed to create native messaging host file: %w", err)
			}
			defer f.Close()

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}

			if err := hostTemplate.Execute(f, map[string]interface{}{
				"ExecPath": execPath,
			}); err != nil {
				return fmt.Errorf("failed to execute template: %w", err)
			}

			if err := os.Chmod(hostPath, 0755); err != nil {
				return fmt.Errorf("failed to make host file executable: %w", err)
			}

			manifestTemplate, err := template.ParseFS(embedFs, "embed/com.github.pomdtr.tweety.json.tmpl")
			if err != nil {
				return fmt.Errorf("failed to parse manifest template: %w", err)
			}

			dirs, err := GetSupportDirs()
			if err != nil {
				return fmt.Errorf("failed to get manifest directories: %w", err)
			}

			for _, dir := range dirs {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					continue
				}

				manifestDir := filepath.Join(dir, "NativeMessagingHosts")
				if err := os.MkdirAll(manifestDir, 0755); err != nil {
					return fmt.Errorf("failed to create native messaging hosts directory: %w", err)
				}

				f, err := os.Create(filepath.Join(manifestDir, "com.github.pomdtr.tweety.json"))
				if err != nil {
					return fmt.Errorf("failed to get manifest file path: %w", err)
				}
				defer f.Close()

				if err := manifestTemplate.Execute(f, map[string]interface{}{
					"Path": hostPath,
				}); err != nil {
					return fmt.Errorf("failed to execute manifest template: %w", err)
				}
			}

			return nil
		},
	}
}
