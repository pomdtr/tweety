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
			res, err := sendMessage(map[string]string{
				"command": "tab.list",
			})
			if err != nil {
				return err
			}

			var tabs []Tab
			if err := json.Unmarshal(res, &tabs); err != nil {
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

	cmd.Flags().Bool("json", false, "output as json")

	return cmd
}

func NewCmdTabPin() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "pin",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.pin",
			}
			if cmd.Flags().Changed("id") {
				id, _ := cmd.Flags().GetInt("id")
				msg["tabId"] = id
			}

			_, err := sendMessage(msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().Int("id", 0, "tab id")
	return cmd
}

func NewCmdTabUnpin() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "unpin",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.unpin",
			}
			if cmd.Flags().Changed("id") {
				id, _ := cmd.Flags().GetInt("id")
				msg["tabId"] = id
			}

			_, err := sendMessage(msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().Int("id", 0, "tab id")
	return cmd
}

func NewCmdTabCreate() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.create",
			}
			if cmd.Flags().Changed("url") {
				_, url := cmd.Flags().GetString("url")
				msg["url"] = url
			}

			_, err := sendMessage(msg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("url", "", "url to open")
	return cmd

}

func NewCmdTabGet(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "get",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.get",
			}

			if cmd.Flags().Changed("id") {
				id, _ := cmd.Flags().GetInt("id")
				msg["tabId"] = id
			}

			res, err := sendMessage(msg)
			if err != nil {
				return err
			}

			var tab Tab
			if err := json.Unmarshal(res, &tab); err != nil {
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

	cmd.Flags().Int("id", 0, "tab id to get")
	cmd.Flags().Bool("json", false, "output as json")

	return cmd
}

func NewCmdTabUrl() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "url",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.get",
			}

			if cmd.Flags().Changed("id") {
				id, _ := cmd.Flags().GetInt("id")
				msg["tabId"] = id
			}

			res, err := sendMessage(msg)
			if err != nil {
				return err
			}

			var tab Tab
			if err := json.Unmarshal(res, &tab); err != nil {
				return err
			}

			fmt.Println(tab.URL)
			return nil
		},
	}

	cmd.Flags().Int("id", 0, "tab id to get url")

	return cmd
}

func NewCmdTabClose() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "close",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.remove",
			}

			if cmd.Flags().Changed("id") {
				ids, _ := cmd.Flags().GetIntSlice("id")
				msg["tabIds"] = ids
			}

			if _, err := sendMessage(msg); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().IntSlice("id", nil, "tab id to close")

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

			if _, err := sendMessage(map[string]any{
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
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{
				"command": "tab.source",
			}

			if cmd.Flags().Changed("id") {
				id, _ := cmd.Flags().GetInt("id")
				msg["tabId"] = id
			}

			res, err := sendMessage(msg)
			if err != nil {
				return err
			}

			var source string
			if err := json.Unmarshal(res, &source); err != nil {
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
		Use: "tab",
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
