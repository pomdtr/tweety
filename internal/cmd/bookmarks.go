package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	"github.com/cli/cli/v2/pkg/jsoncolor"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdBookmarks() *cobra.Command {
	cmd := &cobra.Command{
		Use: "bookmarks",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if env := os.Getenv("TWEETY_SOCKET"); env == "" {
				return fmt.Errorf("TWEETY_SOCKET environment variable must be set")
			}

			return nil
		},
		Short: "Manage bookmarks",
	}

	cmd.AddCommand(
		NewCmdBookmarksGetTree(),
		NewCmdBookmarksGetRecent(),
		NewCmdBookmarksSearch(),
		NewCmdBookmarksCreate(),
		NewCmdBookmarksUpdate(),
		NewCmdBookmarksRemove(),
	)

	return cmd
}

func NewCmdBookmarksGetTree() *cobra.Command {
	return &cobra.Command{
		Use:   "get-tree",
		Short: "Get bookmarks tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "bookmarks.getTree", []any{})
			if err != nil {
				return err
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.Write(resp.Result)
				return nil
			}

			jsoncolor.Write(os.Stdout, bytes.NewReader(resp.Result), "  ")
			return nil
		},
	}
}

func NewCmdBookmarksGetRecent() *cobra.Command {
	return &cobra.Command{
		Use:   "get-recent <number-of-items>",
		Short: "Get recent bookmarks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			numItems, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid number of items: %w", err)
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "bookmarks.getRecent", []any{numItems})
			if err != nil {
				return err
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.Write(resp.Result)
				return nil
			}

			jsoncolor.Write(os.Stdout, bytes.NewReader(resp.Result), "  ")
			return nil
		},
	}
}

func NewCmdBookmarksSearch() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search bookmarks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "bookmarks.search", []any{args[0]})
			if err != nil {
				return err
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.Write(resp.Result)
				return nil
			}

			jsoncolor.Write(os.Stdout, bytes.NewReader(resp.Result), "  ")
			return nil
		},
	}
}

func NewCmdBookmarksCreate() *cobra.Command {
	var flags struct {
		parentId string
		title    string
		url      string
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new bookmark",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.title == "" {
				return fmt.Errorf("title is required")
			}

			bookmark := map[string]interface{}{
				"title": flags.title,
			}

			if flags.parentId != "" {
				bookmark["parentId"] = flags.parentId
			}

			if flags.url != "" {
				bookmark["url"] = flags.url
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "bookmarks.create", []any{bookmark})
			if err != nil {
				return err
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.Write(resp.Result)
				return nil
			}

			jsoncolor.Write(os.Stdout, bytes.NewReader(resp.Result), "  ")
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.parentId, "parent-id", "", "Parent folder ID")
	cmd.Flags().StringVar(&flags.title, "title", "", "Bookmark title (required)")
	cmd.Flags().StringVar(&flags.url, "url", "", "Bookmark URL")
	cmd.MarkFlagRequired("title")

	return cmd
}

func NewCmdBookmarksUpdate() *cobra.Command {
	var flags struct {
		title string
		url   string
	}

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			changes := map[string]interface{}{}

			if flags.title != "" {
				changes["title"] = flags.title
			}

			if flags.url != "" {
				changes["url"] = flags.url
			}

			if len(changes) == 0 {
				return fmt.Errorf("at least one field must be provided to update")
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "bookmarks.update", []any{args[0], changes})
			if err != nil {
				return err
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.Write(resp.Result)
				return nil
			}

			jsoncolor.Write(os.Stdout, bytes.NewReader(resp.Result), "  ")
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.title, "title", "", "New bookmark title")
	cmd.Flags().StringVar(&flags.url, "url", "", "New bookmark URL")

	return cmd
}

func NewCmdBookmarksRemove() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "bookmarks.remove", []any{args[0]})
			if err != nil {
				return err
			}

			return nil
		},
	}
}
