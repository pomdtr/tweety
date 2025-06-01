package cmd

import (
	"fmt"
	"os"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdHistory() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Manage browser history",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if env := os.Getenv("TWEETY_SOCKET"); env == "" {
				return fmt.Errorf("TWEETY_SOCKET environment variable must be set")
			}

			return nil
		},
	}

	cmd.AddCommand(
		NewCmdHistorySearch(),
		NewCmdHistoryAdd(),
		NewCmdHistoryRemove(),
	)

	return cmd
}

func NewCmdHistorySearch() *cobra.Command {
	var flags struct {
		Text string `json:"text"`
	}

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search browser history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := map[string]any{
				"text": flags.Text,
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "history.search", []any{
				query,
			})
			if err != nil {
				return fmt.Errorf("failed to search history: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Text, "text", "t", "", "Text to search in history")
	cmd.MarkFlagRequired("text")

	return cmd
}

func NewCmdHistoryAdd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add entry to browser history",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, _ := cmd.Flags().GetString("url")
			title, _ := cmd.Flags().GetString("title")

			params := map[string]interface{}{"url": url, "title": title}
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "history.add", params)
			if err != nil {
				return fmt.Errorf("failed to add history entry: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().StringP("url", "u", "", "URL to add to history")
	cmd.Flags().StringP("title", "t", "", "Title for the history entry")
	cmd.MarkFlagRequired("url")

	return cmd
}

func NewCmdHistoryRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove entry from browser history",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, _ := cmd.Flags().GetString("url")

			params := map[string]interface{}{"url": url}
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "history.remove", params)
			if err != nil {
				return fmt.Errorf("failed to remove history entry: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().StringP("url", "u", "", "URL to remove from history")
	cmd.MarkFlagRequired("url")

	return cmd
}
