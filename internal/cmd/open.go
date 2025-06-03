package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

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
				fp, err := filepath.Abs(args[0])
				if err != nil {
					return fmt.Errorf("failed to get absolute path: %w", err)
				}

				url := url.URL{
					Scheme: "file",
					Path:   fp,
				}
				options = map[string]string{
					"url": url.String(),
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
