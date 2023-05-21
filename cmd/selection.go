package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

func NewCmdSelection() *cobra.Command {
	cmd := &cobra.Command{
		Use: "selection",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]string{
				"command": "selection.get",
			})
			if err != nil {
				return err
			}

			var selection string
			if err := json.Unmarshal(res, &selection); err != nil {
				return err
			}

			if _, err := os.Stdout.WriteString(selection); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd

}
