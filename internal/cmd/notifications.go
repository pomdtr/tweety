package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/cli/cli/v2/pkg/jsoncolor"
	"github.com/mattn/go-isatty"
	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdNotifications() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notifications",
		Short:   "Manage notifications",
		Aliases: []string{"notification"},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if env := os.Getenv("TWEETY_SOCKET"); env == "" {
				return fmt.Errorf("TWEETY_SOCKET environment variable must be set")
			}
			return nil
		},
	}

	cmd.AddCommand(
		NewCmdNotificationsCreate(),
	)

	return cmd
}

func NewCmdNotificationsCreate() *cobra.Command {
	var options struct {
		Type    string `json:"type"`
		Title   string `json:"title"`
		Message string `json:"message"`
		IconURL string `json:"iconUrl"`
	}
	cmd := &cobra.Command{
		Use:   "create [notification-id]",
		Short: "Create a new browser notification",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var requestArgs []any
			if len(args) > 0 {
				requestArgs = append(requestArgs, args[0])
			}

			requestArgs = append(requestArgs, options)
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "notifications.create", requestArgs)
			if err != nil {
				return fmt.Errorf("failed to create notification: %w", err)
			}

			if !isatty.IsTerminal(os.Stdout.Fd()) {
				os.Stdout.Write(resp.Result)
				return nil
			}

			jsoncolor.Write(os.Stdout, bytes.NewReader(resp.Result), "  ")
			return nil
		},
	}

	cmd.Flags().StringVar(&options.Type, "type", "basic", "Type of notification (basic, image, list, progress)")
	cmd.Flags().StringVar(&options.Title, "title", "", "Title of the notification")
	cmd.MarkFlagRequired("title")
	cmd.Flags().StringVar(&options.Message, "message", "", "Message of the notification")
	cmd.MarkFlagRequired("message")
	cmd.Flags().StringVar(&options.IconURL, "icon-url", "/icons/icon128.png", "URL of the icon for the notification")

	return cmd
}
