package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

var (
	//go:embed manifest.json
	manifest []byte
	//go:embed entrypoint.sh.gotmpl
	entrypoint []byte
)

var (
	manifestTmpl   = template.Must(template.New("manifest").Parse(string(manifest)))
	entrypointTmpl = template.Must(template.New("entrypoint").Parse(string(entrypoint)))
	manifestDirs   = []string{
		filepath.Join(xdg.DataHome, "Google", "Chrome", "NativeMessagingHosts"),
		filepath.Join(xdg.DataHome, "Google", "Chrome Beta", "NativeMessagingHosts"),
		filepath.Join(xdg.DataHome, "microsoft", "edge", "NativeMessagingHosts"),
		filepath.Join(xdg.DataHome, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"),
		filepath.Join(xdg.DataHome, "vivaldi", "NativeMessagingHosts"),
		filepath.Join(xdg.DataHome, "Orion", "NativeMessagingHosts"),
	}
)

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <extension-id>",
		Short: "Init configuration for a browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("unable to get user home directory: %w", err)
			}

			for _, manifestDir := range manifestDirs {
				if _, err := os.Stat(manifestDir); err != nil {
					continue
				}

				manifestBuffer := bytes.Buffer{}
				if err := manifestTmpl.Execute(&manifestBuffer, map[string]string{
					"homeDir":     homeDir,
					"extensionID": args[0],
				}); err != nil {
					return fmt.Errorf("unable to execute manifest template: %w", err)
				}

				manifestPath := filepath.Join(manifestDir, "com.pomdtr.popcorn.json")
				if err := os.WriteFile(manifestPath, manifestBuffer.Bytes(), 0644); err != nil {
					return fmt.Errorf("unable to write manifest file: %w", err)
				}
				cmd.Printf("Manifest file written successfully to %s\n", manifestPath)
			}

			if err := os.MkdirAll(filepath.Join(homeDir, ".local", "bin"), 0755); err != nil {
				return fmt.Errorf("unable to create entrypoint directory: %w", err)
			}

			entrypointBuffer := bytes.Buffer{}
			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("unable to get executable path: %w", err)
			}
			if err := entrypointTmpl.Execute(&entrypointBuffer, map[string]string{
				"popcornBin": execPath,
			}); err != nil {
				return fmt.Errorf("unable to execute entrypoint template: %w", err)
			}

			entrypointPath := filepath.Join(homeDir, ".local", "bin", "popcorn.sh")
			cmd.Printf("Writing entrypoint file to %s\n", entrypointPath)
			if err := os.WriteFile(entrypointPath, entrypointBuffer.Bytes(), 0755); err != nil {
				return fmt.Errorf("unable to write entrypoint file: %w", err)
			}
			cmd.Printf("Entrypoint file written successfully\n")

			return nil
		},
	}

	return cmd
}
