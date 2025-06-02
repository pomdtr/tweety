package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdConfig() *cobra.Command {
	return &cobra.Command{
		Use: "config",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := filepath.Join(configDir, "config.json")
			options := map[string]string{
				"url": fmt.Sprintf("/term.html?mode=editor&file=%s", url.QueryEscape(configPath)),
			}

			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
			if err != nil {
				return fmt.Errorf("failed to create tab: %w", err)
			}

			return nil
		},
	}
}
