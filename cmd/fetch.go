package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

func NewCmdFetch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Do a request from the browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern, _ := cmd.Flags().GetString("pattern")
			res, err := sendMessage[any](map[string]any{
				"command": "fetch",
				"url":     args[0],
				"pattern": pattern,
			})
			if err != nil {
				return err
			}

			encoder := json.NewEncoder(os.Stdout)
			if err := encoder.Encode(res); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("pattern", "", "url pattern of the tab to run the request in")
	return cmd
}
