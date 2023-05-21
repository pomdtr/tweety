package cmd

import (
	"encoding/json"
	"os"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/spf13/cobra"
)

type BitwardenExtension struct {
	Description     string   `json:"description"`
	Enabled         bool     `json:"enabled"`
	HomepageURL     string   `json:"homepageUrl"`
	HostPermissions []string `json:"hostPermissions"`
	Icons           []struct {
		Size int    `json:"size"`
		URL  string `json:"url"`
	} `json:"icons"`
	ID             string   `json:"id"`
	InstallType    string   `json:"installType"`
	IsApp          bool     `json:"isApp"`
	MayDisable     bool     `json:"mayDisable"`
	Name           string   `json:"name"`
	OfflineEnabled bool     `json:"offlineEnabled"`
	OptionsURL     string   `json:"optionsUrl"`
	Permissions    []string `json:"permissions"`
	ShortName      string   `json:"shortName"`
	Type           string   `json:"type"`
	UpdateURL      string   `json:"updateUrl"`
	Version        string   `json:"version"`
}

func NewCmdExtensionList(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			openInBrowser, _ := cmd.Flags().GetBool("open")
			if openInBrowser {
				_, err := sendMessage(map[string]string{
					"command": "tab.create",
					"url":     "chrome://extensions",
				})

				if err != nil {
					return err
				}

				return nil
			}

			res, err := sendMessage(map[string]string{
				"command": "extension.list",
			})
			if err != nil {
				return err
			}

			var extensions []BitwardenExtension
			if err := json.Unmarshal(res, &extensions); err != nil {
				return err
			}

			outputJSON, _ := cmd.Flags().GetBool("json")
			if outputJSON {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(extensions); err != nil {
					return err
				}
				return nil
			}

			for _, extension := range extensions {
				printer.AddField(extension.Name)
				printer.AddField(extension.Version)
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return err
			}

			return nil

		},
	}

	cmd.Flags().Bool("open", false, "open the extension page")
	cmd.Flags().Bool("json", false, "output as JSON")

	return cmd
}

func NewCmdExtension(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use: "extension",
	}

	cmd.AddCommand(NewCmdExtensionList(printer))

	return cmd
}
