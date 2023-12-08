package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	_ "embed"

	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
)

//go:embed extension/com.pomdtr.tweety.json
var manifest []byte

func NewCmdManifest() *cobra.Command {
	var flags struct {
		extensionID string
	}

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Print the manifest for Tweety",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tmpl := template.New("manifest")
			tmpl.Parse(string(manifest))

			execPath, err := os.Executable()
			if err != nil {
				return err
			}

			return tmpl.Execute(os.Stdout, map[string]interface{}{
				"ExecPath":    execPath,
				"ExtensionID": flags.extensionID,
			})
		},
	}

	cmd.Flags().StringVarP(&flags.extensionID, "extension-id", "e", "com.pomdtr.tweety", "extension id")
	cmd.MarkFlagRequired("extension-id")
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
			if len(args) == 1 {
				if !strings.HasPrefix(args[0], "chrome-extension://") {
					return fmt.Errorf("invalid extension id: %s", args[0])
				}

				cmd.PrintErrln("Checking if Tweety is already running...")
				// if another instance of tweety is already running, we don't need to start another one
				for {
					// check if tweety is already running
					resp, err := http.Get("http://localhost:9999/ping")
					if err != nil {
						break
					}

					if resp.StatusCode != http.StatusOK {
						break
					}

					cmd.PrintErrln("Tweety is already running, sleeping for 5 seconds...")
					// sleep for 5 seconds
					time.Sleep(5 * time.Second)
				}

				cmd.PrintErrln("Tweety is not running, starting...")
			}

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

			browserUrl, _ := url.Parse("https://local.tweety.sh")
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

	cmd.AddCommand(NewCmdManifest())
	cmd.Flags().StringVarP(&flags.host, "host", "H", "localhost", "host to listen on")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 9999, "port to listen on")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
