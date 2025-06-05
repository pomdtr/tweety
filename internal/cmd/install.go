package cmd

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed all:embed
var embedFs embed.FS

type BrowserType string

var (
	BrowserTypeChromium BrowserType = "chromium"
	BrowserTypeGecko    BrowserType = "gecko"
)

func NewCmdInstall() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
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

			browsers, err := GetBrowsers()
			if err != nil {
				return fmt.Errorf("failed to get manifest directories: %w", err)
			}

			for _, browser := range browsers {
				if _, err := os.Stat(browser.Dir); os.IsNotExist(err) {
					continue
				}

				manifestDir := filepath.Join(browser.Dir, "NativeMessagingHosts")
				if err := os.MkdirAll(manifestDir, 0755); err != nil {
					return fmt.Errorf("failed to create native messaging hosts directory: %w", err)
				}

				f, err := os.Create(filepath.Join(manifestDir, "com.github.pomdtr.tweety.json"))
				if err != nil {
					return fmt.Errorf("failed to get manifest file path: %w", err)
				}
				defer f.Close()

				if err := manifestTemplate.Execute(f, map[string]interface{}{
					"Path":    hostPath,
					"Browser": browser.Type,
				}); err != nil {
					return fmt.Errorf("failed to execute manifest template: %w", err)
				}
			}

			return nil
		},
	}
}

type Browser struct {
	Dir  string
	Type BrowserType
}

func GetBrowsers() ([]Browser, error) {
	switch runtime.GOOS {
	case "darwin":
		supportDir := filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
		return []Browser{
			{filepath.Join(supportDir, "Google", "Chrome"), BrowserTypeChromium},
			{filepath.Join(supportDir, "Chromium"), BrowserTypeChromium},
			{filepath.Join(supportDir, "BraveSoftware", "Brave-Browser"), BrowserTypeChromium},
			{filepath.Join(supportDir, "Vivaldi"), BrowserTypeChromium},
			{filepath.Join(supportDir, "Microsoft", "Edge"), BrowserTypeChromium},
			{filepath.Join(supportDir, "Mozilla"), BrowserTypeGecko},
			{filepath.Join(supportDir, "zen"), BrowserTypeGecko},
		}, nil
	case "linux":
		configDir := filepath.Join(os.Getenv("HOME"), ".config")
		return []Browser{
			{filepath.Join(configDir, "google-chrome"), BrowserTypeChromium},
			{filepath.Join(configDir, "chromium"), BrowserTypeChromium},
			{filepath.Join(configDir, "microsoft-edge"), BrowserTypeChromium},
		}, nil
	}

	return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
}
