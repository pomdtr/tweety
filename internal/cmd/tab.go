package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/spf13/cobra"
)

type Tab struct {
	Active          bool   `json:"active"`
	Audible         bool   `json:"audible"`
	AutoDiscardable bool   `json:"autoDiscardable"`
	Discarded       bool   `json:"discarded"`
	FavIconURL      string `json:"favIconUrl"`
	GroupID         int    `json:"groupId"`
	Height          int    `json:"height"`
	Highlighted     bool   `json:"highlighted"`
	ID              int    `json:"id"`
	Incognito       bool   `json:"incognito"`
	Index           int    `json:"index"`
	MutedInfo       struct {
		Muted bool `json:"muted"`
	} `json:"mutedInfo"`
	Pinned   bool   `json:"pinned"`
	Selected bool   `json:"selected"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	WindowID int    `json:"windowId"`
}

func NewCmdTabList(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.list",
			}

			windowID, _ := cmd.Flags().GetString("window")
			if windowID != "" {
				id, err := strconv.Atoi(windowID)
				if err != nil {
					return fmt.Errorf("invalid window id: %w", err)
				}

				msg["windowId"] = id
			}

			allWindows, _ := cmd.Flags().GetBool("all")
			if allWindows {
				msg["allWindows"] = true
			}

			tabs, err := sendMessage[[]Tab](msg)
			if err != nil {
				return err
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(tabs); err != nil {
					return err
				}
				return nil
			}

			for _, tab := range tabs {
				printer.AddField(strconv.Itoa(tab.ID))
				printer.AddField(tab.Title)
				printer.AddField(tab.URL)
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringP("window", "w", "", "window id")
	cmd.Flags().Bool("json", false, "output as json")
	cmd.Flags().BoolP("all", "a", false, "list tabs from all windows")

	return cmd
}

func NewCmdTabPin() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "pin",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.pin",
			}

			if len(args) > 0 {
				tabIds := make([]int, len(args))
				for i, arg := range args {
					id, err := strconv.Atoi(arg)
					if err != nil {
						return fmt.Errorf("invalid tab id: %w", err)
					}
					tabIds[i] = id
				}

				msg["tabIds"] = tabIds
			}

			_, err := sendMessage[any](msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func NewCmdTabUnpin() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "unpin",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.unpin",
			}

			if len(args) > 0 {
				tabIds := make([]int, len(args))
				for i, arg := range args {
					id, err := strconv.Atoi(arg)
					if err != nil {
						return fmt.Errorf("invalid tab id: %w", err)
					}
					tabIds[i] = id
				}

				msg["tabIds"] = tabIds
			}

			_, err := sendMessage[any](msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func NewCmdTabCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.create",
			}

			if len(args) > 0 {
				msg["urls"] = args
			}

			_, err := sendMessage[any](msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return cmd

}

func NewCmdTabGet(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "get",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.get",
			}

			if len(args) > 0 {
				tabId, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid tab id: %w", err)
				}

				msg["tabId"] = tabId
			}

			tab, err := sendMessage[Tab](msg)
			if err != nil {
				return err
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(tab); err != nil {
					return err
				}
				return nil
			}

			printer.AddField(strconv.Itoa(tab.ID))
			printer.AddField(tab.Title)
			printer.AddField(tab.URL)
			printer.EndRow()

			if err := printer.Render(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "output as json")

	return cmd
}

func NewCmdTabUrl() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "url",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.get",
			}

			if len(args) > 0 {
				tabId, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid tab id: %w", err)
				}

				msg["tabId"] = tabId
			}

			tab, err := sendMessage[Tab](msg)
			if err != nil {
				return err
			}

			fmt.Println(tab.URL)
			return nil
		},
	}

	return cmd
}

func NewCmdTabClose() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "close",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.remove",
			}

			if len(args) > 0 {
				tabIds := make([]int, len(args))
				for i, arg := range args {
					id, err := strconv.Atoi(arg)
					if err != nil {
						return fmt.Errorf("invalid tab id: %w", err)
					}
					tabIds[i] = id
				}

				msg["tabIds"] = tabIds
			}

			if _, err := sendMessage[any](msg); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func NewCmdTabFocus() *cobra.Command {
	return &cobra.Command{
		Use:  "focus",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabId, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}

			if _, err := sendMessage[any](map[string]any{
				"command": "tab.focus",
				"tabId":   tabId,
			}); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdTabSource() *cobra.Command {
	return &cobra.Command{
		Use:  "source",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.source",
			}

			if len(args) > 0 {
				tabId, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid tab id: %w", err)
				}

				msg["tabId"] = tabId
			}

			source, err := sendMessage[string](msg)
			if err != nil {
				return err
			}

			if _, err := os.Stdout.WriteString(source); err != nil {
				return err
			}
			return nil
		},
	}
}

func NewCmdTab(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tab",
		Short: "Manage Tabs",
	}

	cmd.AddCommand(NewCmdTabList(printer))
	cmd.AddCommand(NewCmdTabFocus())
	cmd.AddCommand(NewCmdTabCreate())
	cmd.AddCommand(NewCmdTabClose())
	cmd.AddCommand(NewCmdTabGet(printer))
	cmd.AddCommand(NewCmdTabUrl())
	cmd.AddCommand(NewCmdTabPin())
	cmd.AddCommand(NewCmdTabUnpin())
	cmd.AddCommand(NewCmdTabSource())

	return cmd
}
