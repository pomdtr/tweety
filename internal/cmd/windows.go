package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdWindows() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "windows",
		Short: "Manage browser windows",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if tweetyPort == 0 || tweetyToken == "" {
				return fmt.Errorf("TWEETY_PORT and TWEETY_TOKEN environment variables must be set")
			}
			return nil

		},
	}

	cmd.AddCommand(
		NewCmdWindowsGetAll(),
		NewCmdWindowsGet(),
		NewCmdWindowsGetCurrent(),
		NewCmdWindowsGetLastFocused(),
		NewCmdWindowsCreate(),
		NewCmdWindowsRemove(),
	)

	return cmd
}

func NewCmdWindowsGetAll() *cobra.Command {
	return &cobra.Command{
		Use:   "get-all",
		Short: "Get all browser windows",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := jsonrpc.NewClient(tweetyPort, tweetyToken)
			resp, err := client.SendRequest("windows.getAll", []any{})
			if err != nil {
				return err
			}
			os.Stdout.Write(resp.Result)
			return nil
		},
	}
}

func NewCmdWindowsGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get <windowID>",
		Short: "Get information about a specific window",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			windowID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid window ID: %w", err)
			}

			client := jsonrpc.NewClient(tweetyPort, tweetyToken)
			resp, err := client.SendRequest("windows.get", []any{windowID})
			if err != nil {
				return err
			}
			os.Stdout.Write(resp.Result)
			return nil
		},
	}
}

func NewCmdWindowsGetCurrent() *cobra.Command {
	return &cobra.Command{
		Use:   "get-current",
		Short: "Get the current window",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := jsonrpc.NewClient(tweetyPort, tweetyToken)
			resp, err := client.SendRequest("windows.getCurrent", []any{})
			if err != nil {
				return err
			}
			os.Stdout.Write(resp.Result)
			return nil
		},
	}
}

func NewCmdWindowsGetLastFocused() *cobra.Command {
	return &cobra.Command{
		Use:   "get-last-focused",
		Short: "Get the last focused window",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := jsonrpc.NewClient(tweetyPort, tweetyToken)
			resp, err := client.SendRequest("windows.getLastFocused", []any{})
			if err != nil {
				return err
			}
			os.Stdout.Write(resp.Result)
			return nil
		},
	}
}

func NewCmdWindowsCreate() *cobra.Command {
	var flags struct {
		url        string
		focused    bool
		incognito  bool
		windowType string
		width      int
		height     int
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new browser window",
		RunE: func(cmd *cobra.Command, args []string) error {
			options := map[string]interface{}{}

			if flags.url != "" {
				options["url"] = flags.url
			}

			if cmd.Flags().Changed("focused") {
				options["focused"] = flags.focused
			}

			if cmd.Flags().Changed("incognito") {
				options["incognito"] = flags.incognito
			}

			if flags.windowType != "" {
				options["type"] = flags.windowType
			}

			if flags.width > 0 {
				options["width"] = flags.width
			}

			if flags.height > 0 {
				options["height"] = flags.height
			}

			client := jsonrpc.NewClient(tweetyPort, tweetyToken)
			resp, err := client.SendRequest("windows.create", []any{options})
			if err != nil {
				return err
			}
			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.url, "url", "", "URL to open in the new window")
	cmd.Flags().BoolVar(&flags.focused, "focused", false, "Focus the new window")
	cmd.Flags().BoolVar(&flags.incognito, "incognito", false, "Open in incognito mode")
	cmd.Flags().StringVar(&flags.windowType, "type", "", "Window type (normal, popup, panel)")
	cmd.Flags().IntVar(&flags.width, "width", 0, "Window width")
	cmd.Flags().IntVar(&flags.height, "height", 0, "Window height")

	return cmd
}

func NewCmdWindowsRemove() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <windowID>",
		Short: "Close a browser window",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			windowID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid window ID: %w", err)
			}

			client := jsonrpc.NewClient(tweetyPort, tweetyToken)
			_, err = client.SendRequest("windows.remove", []any{windowID})
			if err != nil {
				return err
			}
			return nil
		},
	}
}
