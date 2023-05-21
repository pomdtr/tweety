package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCmdBookmarkList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]string{
				"command": "bookmark.list",
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

func NewCmdBookMark() *cobra.Command {
	cmd := &cobra.Command{
		Use: "bookmark",
	}

	cmd.AddCommand(NewCmdBookmarkList())

	return cmd
}
