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

const manifestName = "com.pomdtr.webterm.json"

var (
	//go:embed manifest.json
	manifest []byte
	//go:embed webterm.sh
	entrypoint []byte
)

var (
	manifestTmpl  = template.Must(template.New("manifest").Parse(string(manifest)))
	manifestPaths = map[string]string{
		"chrome":  filepath.Join(xdg.DataHome, "Google", "Chrome", "NativeMessagingHosts", manifestName),
		"edge":    filepath.Join(xdg.DataHome, "microsoft", "edge", "NativeMessagingHosts", manifestName),
		"brave":   filepath.Join(xdg.DataHome, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts", manifestName),
		"vivaldi": filepath.Join(xdg.DataHome, "vivaldi", "NativeMessagingHosts", manifestName),
		"arc":     filepath.Join(xdg.DataHome, "Arc", "User Data", "NativeMessagingHosts", manifestName),
	}
)

func NewCmdInit() *cobra.Command {
	flags := struct {
		Browser     string
		ExtensionID string
	}{}

	cmd := &cobra.Command{
		Use: "init <browser>",
		RunE: func(cmd *cobra.Command, args []string) error {

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("unable to get user home directory: %w", err)
			}

			manifestPath, ok := manifestPaths[flags.Browser]
			if !ok {
				return fmt.Errorf("invalid browser: %s", flags.Browser)
			}

			cmd.Printf("Writing manifest file to %s\n", manifestPath)
			if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
				return fmt.Errorf("unable to create manifest directory: %w", err)
			}

			writer := bytes.Buffer{}
			if err := manifestTmpl.Execute(&writer, map[string]string{
				"homeDir":     homeDir,
				"extensionID": flags.ExtensionID,
			}); err != nil {
				return fmt.Errorf("unable to execute manifest template: %w", err)
			}

			if err := os.WriteFile(manifestPath, writer.Bytes(), 0644); err != nil {
				return fmt.Errorf("unable to write manifest file: %w", err)
			}
			cmd.Printf("Manifest file written successfully\n")

			entrypointPath := filepath.Join(homeDir, ".local", "bin", "webterm.sh")
			cmd.Printf("Writing entrypoint file to %s\n", entrypointPath)
			if err := os.WriteFile(entrypointPath, []byte(entrypoint), 0755); err != nil {
				return fmt.Errorf("unable to write entrypoint file: %w", err)
			}
			cmd.Printf("Entrypoint file written successfully\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.Browser, "browser", "", "Browser to install the extension for")
	cmd.MarkFlagRequired("browser")
	cmd.Flags().StringVar(&flags.ExtensionID, "extension-id", "", "Extension ID to install")
	cmd.MarkFlagRequired("extension-id")

	return cmd
}
