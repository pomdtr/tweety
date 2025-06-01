package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdTabs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tabs",
		Short: "Manage tabs",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if env := os.Getenv("TWEETY_SOCKET"); env == "" {
				return fmt.Errorf("TWEETY_SOCKET environment variable must be set")
			}

			return nil
		},
	}

	cmd.AddCommand(
		NewCmdTabQuery(),
		NewCmdTabsGet(),
		NewCmdTabsCreate(),
		NewCmdTabsDuplicate(),
		NewCmdTabsDiscard(),
		NewCmdTabsRemove(),
		NewCmdTabsUpdate(),
		NewCmdTabsReload(),
		NewCmdTabsGoForward(),
		NewCmdTabsGoBack(),
		NewCmdTabsCaptureVisibleTab(),
	)

	return cmd
}

func NewCmdTabQuery() *cobra.Command {
	var flags struct {
		Active            bool
		Pinned            bool
		Highlighted       bool
		LastFocusedWindow bool
	}

	cmd := &cobra.Command{
		Use:   "query",
		Short: "List all tabs",
		RunE: func(cmd *cobra.Command, args []string) error {
			options := make(map[string]any)
			if cmd.Flags().Changed("active") {
				options["active"] = flags.Active
			}

			if cmd.Flags().Changed("pinned") {
				options["pinned"] = flags.Pinned
			}

			if cmd.Flags().Changed("highlighted") {
				options["highlighted"] = flags.Highlighted
			}

			if cmd.Flags().Changed("last-focused-window") {
				options["lastFocusedWindow"] = flags.LastFocusedWindow
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.query", []any{
				options,
			})
			if err != nil {
				return fmt.Errorf("failed to list tabs: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.Active, "active", false, "Filter active tabs")
	cmd.Flags().BoolVar(&flags.Pinned, "pinned", false, "Filter pinned tabs")
	cmd.Flags().BoolVar(&flags.Highlighted, "highlighted", false, "Filter highlighted tabs")
	cmd.Flags().BoolVar(&flags.LastFocusedWindow, "last-focused-window", false, "Filter tabs in the last focused window")

	return cmd
}

func NewCmdTabsCreate() *cobra.Command {
	var flags struct {
		URL    string
		Pinned bool
		Active bool
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new tab",
		RunE: func(cmd *cobra.Command, args []string) error {
			options := map[string]any{
				"active": true,
			}

			if cmd.Flags().Changed("url") {
				options["url"] = flags.URL
			}

			if cmd.Flags().Changed("pinned") {
				options["pinned"] = flags.Pinned
			}

			if cmd.Flags().Changed("active") {
				options["active"] = flags.Active
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
			if err != nil {
				return fmt.Errorf("failed to create tab: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.URL, "url", "", "URL to open in the new tab")
	cmd.Flags().BoolVar(&flags.Pinned, "pinned", false, "Pin the new tab")
	cmd.Flags().BoolVar(&flags.Active, "active", false, "Activate the new tab")

	return cmd
}

func NewCmdTabsRemove() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove <tabID> [<tabID>...]",
		Short:             "Close a tab",
		ValidArgsFunction: completeTabID,
		Args:              cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var tabIds []int
			for _, arg := range args {
				tabID, err := strconv.Atoi(arg)
				if err != nil {
					return fmt.Errorf("invalid tab ID '%s': %w", arg, err)
				}

				tabIds = append(tabIds, tabID)
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.remove", []any{tabIds})
			if err != nil {
				return fmt.Errorf("failed to close tab: %w", err)
			}

			if resp.Error != nil {
				os.Stderr.Write(resp.Error)
				os.Exit(1)
			}

			return nil
		},
	}

	return cmd
}

func NewCmdTabsUpdate() *cobra.Command {
	var flags struct {
		URL         string `json:"url"`
		Active      bool   `json:"active"`
		Highlighted bool   `json:"highlighted"`
		Pinned      bool   `json:"pinned"`
		Muted       bool   `json:"muted"`
	}

	cmd := &cobra.Command{
		Use:   "update <tabID>",
		Short: "Focus a tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid tab ID: %w", err)
			}

			options := make(map[string]any)
			if cmd.Flags().Changed("url") {
				options["url"] = flags.URL
			}

			if cmd.Flags().Changed("active") {
				options["active"] = flags.Active
			}

			if cmd.Flags().Changed("highlighted") {
				options["highlighted"] = flags.Highlighted
			}

			if cmd.Flags().Changed("pinned") {
				options["pinned"] = flags.Pinned
			}

			if cmd.Flags().Changed("muted") {
				options["muted"] = flags.Muted
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.update", []any{
				tabID,
				options,
			})
			if err != nil {
				return fmt.Errorf("failed to focus tab: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.URL, "url", "", "URL to update in the tab")
	cmd.Flags().BoolVar(&flags.Active, "active", false, "Activate the tab")
	cmd.Flags().BoolVar(&flags.Highlighted, "highlighted", false, "Highlight the tab")
	cmd.Flags().BoolVar(&flags.Pinned, "pinned", false, "Pin the tab")
	cmd.Flags().BoolVar(&flags.Muted, "muted", false, "Mute the tab")

	return cmd
}

func NewCmdTabsGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get [tabID]",
		ValidArgsFunction: completeTabID,
		Short:             "Get information about a specific tab",
		Args:              cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := []any{}
			if len(args) > 0 {
				tabID, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid tab ID: %w", err)
				}
				params = append(params, tabID)
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.get", params)
			if err != nil {
				return fmt.Errorf("failed to get tab: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	return cmd
}

func NewCmdTabsDuplicate() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "duplicate <tabID>",
		Short:             "Duplicate a tab",
		ValidArgsFunction: completeTabID,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid tab ID: %w", err)
			}

			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.duplicate", []any{tabID})
			if err != nil {
				return fmt.Errorf("failed to duplicate tab: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	return cmd
}

func NewCmdTabsDiscard() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "discard <tabID> [<tabID>...]",
		Short:             "Discard (unload) tabs to free memory",
		ValidArgsFunction: completeTabID,
		Args:              cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var tabIds []int
			for _, arg := range args {
				tabID, err := strconv.Atoi(arg)
				if err != nil {
					return fmt.Errorf("invalid tab ID '%s': %w", arg, err)
				}

				tabIds = append(tabIds, tabID)
			}

			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.discard", []any{tabIds})
			if err != nil {
				return fmt.Errorf("failed to discard tabs: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func completeTabID(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.query", []any{map[string]any{}})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var tabs []map[string]any
	if err := json.Unmarshal(resp.Result, &tabs); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, tab := range tabs {
		tabID, ok := tab["id"].(int)
		if !ok {
			continue
		}

		completions = append(completions, strconv.Itoa(tabID))
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func NewCmdTabsCaptureVisibleTab() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capture-visible-tab",
		Short: "Capture the visible area of a tab",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.captureVisibleTab", []any{})
			if err != nil {
				return fmt.Errorf("failed to capture visible tab: %w", err)
			}

			os.Stdout.Write(resp.Result)
			return nil
		},
	}

	return cmd
}

func NewCmdTabsReload() *cobra.Command {
	var flags struct {
		BypassCache bool
	}

	cmd := &cobra.Command{
		Use:               "reload <tabID>",
		Short:             "Reload a tab",
		ValidArgsFunction: completeTabID,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid tab ID: %w", err)
			}

			var reloadProperties map[string]any
			if cmd.Flags().Changed("bypass-cache") {
				reloadProperties = map[string]any{
					"bypassCache": flags.BypassCache,
				}
			}

			_, err = jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.reload", []any{tabID, reloadProperties})
			if err != nil {
				return fmt.Errorf("failed to reload tab: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.BypassCache, "bypass-cache", false, "Bypass cache when reloading")

	return cmd
}

func NewCmdTabsGoForward() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "go-forward <tabID>",
		Short:             "Navigate tab forward in history",
		ValidArgsFunction: completeTabID,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid tab ID: %w", err)
			}

			_, err = jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.goForward", []any{tabID})
			if err != nil {
				return fmt.Errorf("failed to navigate tab forward: %w", err)
			}

			return nil
		},
	}

	return cmd
}

func NewCmdTabsGoBack() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "go-back <tabID>",
		Short:             "Navigate tab backward in history",
		ValidArgsFunction: completeTabID,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid tab ID: %w", err)
			}

			_, err = jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.goBackward", []any{tabID})
			if err != nil {
				return fmt.Errorf("failed to navigate tab backward: %w", err)
			}

			return nil
		},
	}

	return cmd
}
