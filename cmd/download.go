package cmd

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/spf13/cobra"
)

type Download struct {
	BytesReceived int    `json:"bytesReceived"`
	CanResume     bool   `json:"canResume"`
	Danger        string `json:"danger"`
	EndTime       string `json:"endTime"`
	Exists        bool   `json:"exists"`
	FileSize      int    `json:"fileSize"`
	Filename      string `json:"filename"`
	FinalURL      string `json:"finalUrl"`
	ID            int    `json:"id"`
	Incognito     bool   `json:"incognito"`
	MIME          string `json:"mime"`
	Paused        bool   `json:"paused"`
	Referrer      string `json:"referrer"`
	StartTime     string `json:"startTime"`
	State         string `json:"state"`
	TotalBytes    int    `json:"totalBytes"`
	URL           string `json:"url"`
}

func NewCmdDownloadList(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			openInBrowser, _ := cmd.Flags().GetBool("web")
			if openInBrowser {
				_, err := sendMessage(map[string]string{
					"command": "tab.create",
					"url":     "chrome://downloads",
				})

				if err != nil {
					return err
				}

				return nil
			}

			res, err := sendMessage(map[string]string{
				"command": "download.list",
			})
			if err != nil {
				return err
			}

			var downloads []Download
			if err := json.Unmarshal(res, &downloads); err != nil {
				return err
			}

			outputJSON, _ := cmd.Flags().GetBool("json")
			if outputJSON {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				if err := encoder.Encode(downloads); err != nil {
					return err
				}
				return nil
			}

			for _, download := range downloads {
				printer.AddField(strconv.Itoa(download.ID))
				printer.AddField(download.Filename)
				printer.AddField(download.State)
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "json output")
	cmd.Flags().Bool("web", false, "open in browser")
	return cmd
}

func NewCmdDownload(printer tableprinter.TablePrinter) *cobra.Command {
	cmd := &cobra.Command{
		Use: "download",
	}

	cmd.AddCommand(NewCmdDownloadList(printer))

	return cmd
}
