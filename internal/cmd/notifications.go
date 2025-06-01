package cmd

import (
	"fmt"
	"os"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdNotifications() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notifications",
		Short: "Manage browser notifications",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if tweetyPort == 0 || tweetyToken == "" {
				return fmt.Errorf("TWEETY_PORT and TWEETY_TOKEN environment variables must be set")
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
			client := jsonrpc.NewClient(tweetyPort, tweetyToken)
			var requestArgs []any
			if len(args) > 0 {
				requestArgs = append(requestArgs, args[0])
			}

			requestArgs = append(requestArgs, options)
			resp, err := client.SendRequest("notifications.create", requestArgs)
			if err != nil {
				return fmt.Errorf("failed to create notification: %w", err)
			}
			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().StringVar(&options.Type, "type", "basic", "Type of notification (basic, image, list, progress)")
	cmd.MarkFlagRequired("type")
	cmd.Flags().StringVar(&options.Title, "title", "", "Title of the notification")
	cmd.MarkFlagRequired("title")
	cmd.Flags().StringVar(&options.Message, "message", "", "Message of the notification")
	cmd.MarkFlagRequired("message")
	cmd.Flags().StringVar(&options.IconURL, "icon-url", "", "URL of the icon for the notification")
	cmd.MarkFlagRequired("icon-url")

	return cmd
}
