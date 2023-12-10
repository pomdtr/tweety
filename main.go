package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"text/template"

	_ "embed"

	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
)

//go:embed com.pomdtr.tweety.plist
var launchdService []byte

func LaunchdService(write io.Writer) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	tmpl := template.New("service")
	tmpl.Parse(string(launchdService))
	return tmpl.Execute(os.Stdout, map[string]interface{}{
		"ExecPath": execPath,
		"HomeDir":  homeDir,
	})
}

//go:embed tweety.service
var systemdService []byte

func SystemdService(write io.Writer) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	tmpl := template.New("service")
	tmpl.Parse(string(systemdService))
	return tmpl.Execute(os.Stdout, map[string]interface{}{
		"ExecPath": execPath,
		"HomeDir":  homeDir,
	})
}

func NewCmdService() *cobra.Command {
	cmd := &cobra.Command{
		Use: "service",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch runtime.GOOS {
			case "darwin":
				return LaunchdService(os.Stdout)
			case "linux":
				return SystemdService(os.Stdout)
			default:
				return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
			}
		},
	}

	return cmd
}

func main() {
	var flags struct {
		host string
		port int
	}
	cmd := cobra.Command{
		Use:          "tweety",
		Short:        "An integrated terminal for your web browser",
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handler, err := NewHandler()
			if err != nil {
				return err
			}

			port := flags.port
			if port == 0 {
				p, err := freeport.GetFreePort()
				if err != nil {
					return err
				}

				port = p
			}

			browserUrl, _ := url.Parse("https://tweety.sh")
			if cmd.Flags().Changed("port") {
				query := browserUrl.Query()
				query.Set("port", fmt.Sprintf("%d", port))
				browserUrl.RawQuery = query.Encode()
			}

			cmd.PrintErrln("Listening on", fmt.Sprintf("http://%s:%d", flags.host, port))
			cmd.PrintErrln("Browser Friendly URL:", browserUrl.String())
			cmd.Println("Press Ctrl+C to exit")
			return http.ListenAndServe(fmt.Sprintf("%s:%d", flags.host, port), handler)
		},
	}

	cmd.Flags().StringVarP(&flags.host, "host", "H", "localhost", "host to listen on")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 9999, "port to listen on")

	cmd.AddCommand(NewCmdService())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
