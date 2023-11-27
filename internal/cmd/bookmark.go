package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

func NewCmdBookmarkList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage[any](map[string]string{
				"command": "bookmark.list",
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
}

func NewCmdBookMark() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookmark",
		Short: "Manage bookmarks",
	}

	cmd.AddCommand(NewCmdBookmarkList())

	return cmd
}
