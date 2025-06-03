package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

func NewCmdRun() *cobra.Command {
	cmd := &cobra.Command{
		Use: "run <app> [args...]",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveDefault
			}

			entries, err := os.ReadDir(appDir)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				// Strip extension for completion
				name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				completions = append(completions, name)
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			files, readErr := os.ReadDir(appDir)
			if readErr != nil {
				return fmt.Errorf("failed to read app directory: %w", readErr)
			}

			for _, file := range files {
				name := file.Name()
				nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))
				if nameWithoutExt != args[0] {
					continue
				}

				entrypoint := filepath.Join(appDir, name)
				// check if the file is executable
				stat, err := os.Stat(entrypoint)
				if err != nil {
					return fmt.Errorf("failed to stat command entrypoint: %w", err)
				}

				if stat.IsDir() {
					continue
				}

				// check if the entrypoint is executable
				if stat.Mode()&0111 == 0 {
					if err := os.Chmod(entrypoint, 0755); err != nil {
						return fmt.Errorf("failed to make command entrypoint executable: %w", err)
					}
				}

				appUrl := url.URL{
					Path: "/term.html",
					RawQuery: url.Values{
						"app": []string{nameWithoutExt},
						"arg": args[1:],
					}.Encode(),
				}

				options := map[string]string{"url": appUrl.String()}
				_, err = jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
				if err != nil {
					return fmt.Errorf("failed to create tab: %w", err)
				}

				return nil
			}

			return fmt.Errorf("unknown app: %s", args[0])
		},
	}

	cmd.Flags().SetInterspersed(false)
	return cmd

}
