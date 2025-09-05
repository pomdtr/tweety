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

func NewCmdOpen() *cobra.Command {
	var flags struct {
		arg []string
	}

	cmd := &cobra.Command{
		Use:  "open <url|file|app>",
		Args: cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveDefault
			}

			entries, err := os.ReadDir(appDir)
			if err != nil {
				return nil, cobra.ShellCompDirectiveDefault
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

			return completions, cobra.ShellCompDirectiveDefault
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
						"arg": flags.arg,
					}.Encode(),
				}

				options := map[string]string{"url": appUrl.String()}
				_, err = jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
				if err != nil {
					return fmt.Errorf("failed to create tab: %w", err)
				}

				return nil
			}

			var options map[string]string
			if _, err := os.Stat(args[0]); err == nil {
				fp, err := filepath.Abs(args[0])
				if err != nil {
					return fmt.Errorf("failed to get absolute path: %w", err)
				}

				url := url.URL{
					Scheme: "file",
					Path:   fp,
				}
				options = map[string]string{
					"url": url.String(),
				}
			} else {
				options = map[string]string{
					"url": args[0],
				}
			}

			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
			if err != nil {
				return fmt.Errorf("failed to create tab: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&flags.arg, "arg", "a", []string{}, "Argument to pass to the app")

	return cmd
}
