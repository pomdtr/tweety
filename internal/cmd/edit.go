package cmd

import (
	"fmt"
	"os"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdEdit() *cobra.Command {
	return &cobra.Command{
		Use:  "edit <file>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options := map[string]string{
				"url": fmt.Sprintf("/tty.html?mode=editor&file=%s", args[0]),
			}

			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
			if err != nil {
				return fmt.Errorf("failed to create tab: %w", err)
			}

			return nil
		},
	}
}
