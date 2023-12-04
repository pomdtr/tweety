package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/phayes/freeport"
	"github.com/spf13/cobra"
)

func main() {
	var flags struct {
		host string
		port int
	}
	cmd := cobra.Command{
		Use:   "tweety",
		Short: "An integrated terminal for your web browser",
		Args:  cobra.NoArgs,
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

			cmd.PrintErrln("Listening on", fmt.Sprintf("http://%s:%d", flags.host, port))
			return http.ListenAndServe(fmt.Sprintf("%s:%d", flags.host, port), handler)
		},
	}

	cmd.Flags().StringVarP(&flags.host, "host", "H", "localhost", "host to listen on")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 9999, "port to listen on")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
