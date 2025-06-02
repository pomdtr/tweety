package cmd

import (
	"fmt"
	"os"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdOpen() *cobra.Command {
	return &cobra.Command{
		Use:  "open <url>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var options map[string]string
			if _, err := os.Stat(args[0]); err == nil {
				options = map[string]string{
					"url": fmt.Sprintf("file://%s", args[0]),
				}
			} else {
				options = map[string]string{
					"url": args[0],
				}
			}

			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
			if err != nil {
				return fmt.Errorf("failed to create tab: %w", err)
			}

			return nil
		},
	}
}
