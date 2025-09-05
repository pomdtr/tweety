package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use: "run <command> [args...]",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			files, err := os.ReadDir(commandDir)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var commands []string
			for _, file := range files {
				if file.IsDir() {
					continue
				}

				name := file.Name()
				// Strip any extension for command completion
				name = strings.TrimSuffix(name, filepath.Ext(name))
				commands = append(commands, name)
			}

			return commands, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// First try to find the exact file name
			entrypoint := filepath.Join(commandDir, args[0])
			stat, err := os.Stat(entrypoint)

			// If not found, try to find any file that starts with the command name
			if os.IsNotExist(err) {
				files, readErr := os.ReadDir(commandDir)
				if readErr == nil {
					for _, file := range files {
						if file.IsDir() {
							continue
						}

						name := file.Name()
						nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))

						if nameWithoutExt == args[0] {
							entrypoint = filepath.Join(commandDir, name)
							stat, err = os.Stat(entrypoint)
							break
						}
					}
				}
			}

			if err != nil {
				return fmt.Errorf("unknown command: %s", args[0])
			}

			if stat.IsDir() {
				return fmt.Errorf("command entrypoint is a directory, expected a file: %s", entrypoint)
			}

			// check if the entrypoint is executable
			if stat.Mode()&0111 == 0 {
				if err := os.Chmod(entrypoint, 0755); err != nil {
					return fmt.Errorf("failed to make command entrypoint executable: %w", err)
				}
			}

			cmdExec := exec.Command(entrypoint, args[1:]...)

			cmdExec.Stdin = os.Stdin
			cmdExec.Stdout = os.Stdout
			cmdExec.Stderr = os.Stderr

			cmd.SilenceErrors = true
			return cmdExec.Run()
		},
	}

	cmd.Flags().SetInterspersed(false)
	return cmd

}
