package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed all:embed
var embedFs embed.FS

func NewCmdInstall() *cobra.Command {
	var flags struct {
		extensionID string
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install native messaging host",
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
					"Path":        hostPath,
					"ExtensionID": flags.extensionID,
				}); err != nil {
					return fmt.Errorf("failed to execute manifest template: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.extensionID, "extension-id", "", "Extension ID for the native messaging host")
	cmd.MarkFlagRequired("extension-id")

	return cmd
}

func NewCmdUninstall() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall native messaging host",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := GetSupportDirs()
			if err != nil {
				return fmt.Errorf("failed to get manifest directories: %w", err)
			}

			hostPath := filepath.Join(dataDir, "native_messaging_host")
			if err := os.Remove(hostPath); err != nil {
				return fmt.Errorf("failed to remove native messaging host file: %w", err)
			}

			for _, dir := range dirs {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					continue
				}

				manifestPath := filepath.Join(dir, "NativeMessagingHosts", "com.github.pomdtr.tweety.json")
				if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove manifest file: %w", err)
				}
			}

			return nil
		},
	}
}
