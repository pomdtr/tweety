package cmd

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/spf13/cobra"
)

type Window struct {
	AlwaysOnTop bool   `json:"alwaysOnTop"`
	Focused     bool   `json:"focused"`
	Height      int    `json:"height"`
	ID          int    `json:"id"`
	Incognito   bool   `json:"incognito"`
	Left        int    `json:"left"`
	State       string `json:"state"`
	Top         int    `json:"top"`
	Type        string `json:"type"`
	Width       int    `json:"width"`
}

func NewCmdWindowList(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]string{
				"command": "window.list",
			})
			if err != nil {
				return err
			}

			var windows []Window
			if err := json.Unmarshal(res, &windows); err != nil {
				return err
			}

			outputJSON, _ := cmd.Flags().GetBool("json")
			if outputJSON {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(windows); err != nil {
					return err
				}

				return nil
			}

			for _, window := range windows {
				printer.AddField(strconv.Itoa(window.ID))
				printer.AddField(strconv.Itoa(window.Width))
				printer.AddField(strconv.Itoa(window.Height))
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "json output")

	return cmd
}
func NewCmdWindow(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use: "window",
	}

	cmd.AddCommand(NewCmdWindowList(printer))

	return cmd
}
