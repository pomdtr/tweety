package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCmdFetch() *cobra.Command {
	return &cobra.Command{
		Use:  "fetch",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]any{
				"command": "fetch",
				"url":     args[0],
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

}
