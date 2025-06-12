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

func NewCmdWindows() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "window",
		Aliases: []string{"windows"},
		Short:   "Manage browser windows",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if env := os.Getenv("TWEETY_SOCKET"); env == "" {
				return fmt.Errorf("TWEETY_SOCKET environment variable must be set")
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
		NewCmdWindowsUpdate(),
		NewCmdWindowsRemove(),
	)

	return cmd
}

func NewCmdWindowsGetAll() *cobra.Command {
	return &cobra.Command{
		Use:   "get-all",
		Short: "Get all browser windows",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "windows.getAll", []any{})
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

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "windows.get", []any{windowID})
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

func NewCmdWindowsGetCurrent() *cobra.Command {
	return &cobra.Command{
		Use:   "get-current",
		Short: "Get the current window",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "windows.getCurrent", []any{})
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

func NewCmdWindowsGetLastFocused() *cobra.Command {
	return &cobra.Command{
		Use:   "get-last-focused",
		Short: "Get the last focused window",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "windows.getLastFocused", []any{})
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

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "windows.create", []any{options})
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

	cmd.Flags().StringVar(&flags.url, "url", "", "URL to open in the new window")
	cmd.Flags().BoolVar(&flags.focused, "focused", false, "Focus the new window")
	cmd.Flags().BoolVar(&flags.incognito, "incognito", false, "Open in incognito mode")
	cmd.Flags().StringVar(&flags.windowType, "type", "", "Window type (normal, popup, panel)")
	cmd.Flags().IntVar(&flags.width, "width", 0, "Window width")
	cmd.Flags().IntVar(&flags.height, "height", 0, "Window height")

	return cmd
}

func NewCmdWindowsUpdate() *cobra.Command {
	var flags struct {
		focused       bool
		state         string
		width         int
		height        int
		left          int
		top           int
		drawAttention bool
	}

	cmd := &cobra.Command{
		Use:   "update <windowID>",
		Short: "Update properties of a browser window",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			windowID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid window ID: %w", err)
			}

			options := make(map[string]interface{})

			if cmd.Flags().Changed("focused") {
				options["focused"] = flags.focused
			}

			if flags.state != "" {
				options["state"] = flags.state
			}

			if flags.width > 0 {
				options["width"] = flags.width
			}

			if flags.height > 0 {
				options["height"] = flags.height
			}

			if cmd.Flags().Changed("left") {
				options["left"] = flags.left
			}

			if cmd.Flags().Changed("top") {
				options["top"] = flags.top
			}

			if cmd.Flags().Changed("draw-attention") {
				options["drawAttention"] = flags.drawAttention
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "windows.update", []any{windowID, options})
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

	cmd.Flags().BoolVar(&flags.focused, "focused", false, "Focus the window")
	cmd.Flags().StringVar(&flags.state, "state", "", "Window state (normal, minimized, maximized, fullscreen)")
	cmd.Flags().IntVar(&flags.width, "width", 0, "Window width")
	cmd.Flags().IntVar(&flags.height, "height", 0, "Window height")
	cmd.Flags().IntVar(&flags.left, "left", 0, "Window left position")
	cmd.Flags().IntVar(&flags.top, "top", 0, "Window top position")
	cmd.Flags().BoolVar(&flags.drawAttention, "draw-attention", false, "Draw attention to the window")

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

			_, err = jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "windows.remove", []any{windowID})
			if err != nil {
				return err
			}
			return nil
		},
	}
}
