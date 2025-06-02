package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

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
				// Print the app name (file name)
				cmd.Println(entry.Name())
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
			entrypoint := filepath.Join(appDir, args[0])
			if _, err := os.Stat(entrypoint); os.IsNotExist(err) {
				return fmt.Errorf("app file %s does not exist", args[0])
			}

			options := map[string]string{
				"url": fmt.Sprintf("/term.html?mode=editor&file=%s", url.QueryEscape(entrypoint)),
			}

			_, err := jsonrpc.SendRequest(os.Getenv("TWEETY_SOCKET"), "tabs.create", []any{options})
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
		completions = append(completions, entry.Name())
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
