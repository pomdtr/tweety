package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdFetch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch <url>",
		Short: "Fetch from service worker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: add flags for headers, method, body, etc...
			options := map[string]any{}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "fetch", []any{args[0], options})
			if err != nil {
				return err
			}

			var res struct {
				Status  int               `json:"status"`
				Headers map[string]string `json:"headers"`
				Body    []byte            `json:"body"`
			}

			if err := json.Unmarshal(resp.Result, &res); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}

			os.Stdout.Write(res.Body)
			return nil
		},
	}

	return cmd
}
