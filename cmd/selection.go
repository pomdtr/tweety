package cmd

import (
	"encoding/json"
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func NewCmdSelection() *cobra.Command {
	cmd := &cobra.Command{
		Use: "selection",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isatty.IsTerminal(os.Stdin.Fd()) {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}

				if _, err := sendMessage(map[string]string{
					"command": "selection.set",
					"text":    string(b),
				}); err != nil {
					return err
				}

				return nil
			}

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
