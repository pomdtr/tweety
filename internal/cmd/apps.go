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

var (
	appDir = filepath.Join(configDir, "apps")
)

func NewCmdApps() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apps",
		Aliases: []string{"app"},
		Short:   "Manage Tweety apps",
		Long:    "This command allows you to manage Tweety apps.",
	}

	cmd.AddCommand(
		NewCmdAppsOpen(),
		NewCmdAppsList(),
		NewCmdAppsEdit(),
	)

	return cmd
}

func NewCmdAppsOpen() *cobra.Command {
	return &cobra.Command{
		Use:               "open <app>",
		Short:             "Open a Tweety app",
		ValidArgsFunction: completeApp,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := args[0]
			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{
				map[string]string{
					"url": "/term.html?mode=app&app=" + url.QueryEscape(appName),
				},
			})

			if err != nil {
				return err
			}
			// Implementation for opening the app goes here
			return nil
		},
	}
}

func NewCmdAppsList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := os.ReadDir(appDir)
			if err != nil {
				return err
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				// Strip extension for display
				name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				cmd.Println(name)
			}
			return nil
		},
	}
}

func NewCmdAppsEdit() *cobra.Command {
	return &cobra.Command{
		Use:               "edit <file>",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeApp,
		RunE: func(cmd *cobra.Command, args []string) error {
			// First try to find the exact file name
			entrypoint := filepath.Join(appDir, args[0])
			_, err := os.Stat(entrypoint)

			// If not found, try to find any file that starts with the app name
			if os.IsNotExist(err) {
				files, readErr := os.ReadDir(appDir)
				if readErr == nil {
					for _, file := range files {
						if file.IsDir() {
							continue
						}

						name := file.Name()
						nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))

						if nameWithoutExt == args[0] {
							entrypoint = filepath.Join(appDir, name)
							_, err = os.Stat(entrypoint)
							break
						}
					}
				}
			}

			if err != nil {
				return fmt.Errorf("app file %s does not exist", args[0])
			}

			options := map[string]string{
				"url": fmt.Sprintf("/term.html?mode=editor&file=%s", url.QueryEscape(entrypoint)),
			}

			_, err = jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
			if err != nil {
				return fmt.Errorf("failed to create tab: %w", err)
			}

			return nil
		},
	}

}

func completeApp(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
}
