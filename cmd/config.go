package cmd

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewCmdConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use: "config",
		RunE: func(cmd *cobra.Command, args []string) error {
			editor, ok := os.LookupEnv("EDITOR")
			if !ok {
				editor = "vim"
			}

			homedir, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			configDir := filepath.Join(homedir, ".config", "popcorn")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return err
			}

			configPath := filepath.Join(homedir, ".config", "popcorn", "config.json")

			command := exec.Command(editor, configPath)
			command.Stdin = os.Stdin
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			if err := command.Run(); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}
