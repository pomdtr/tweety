package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCmdFetch() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "fetch",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern, _ := cmd.Flags().GetString("pattern")
			res, err := sendMessage(map[string]any{
				"command": "fetch",
				"url":     args[0],
				"pattern": pattern,
			})
			if err != nil {
				return err
			}

			if _, err := os.Stdout.Write(res); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("pattern", "", "url pattern of the tab to run the request in")
	return cmd
}
