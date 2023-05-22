package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	_ "embed"

	"github.com/adrg/xdg"
	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const webtermPort = 9999

var (
	//go:embed manifest.json
	manifest []byte
	//go:embed webterm.sh
	entrypoint []byte
)

func sendMessage(payload any) ([]byte, error) {
	target := fmt.Sprintf("http://localhost:%d/browser", webtermPort)
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	res, err := http.Post(target, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf(string(msg))
	}

	return io.ReadAll(res.Body)
}

func NewCmdInit() *cobra.Command {
	cmd := &cobra.Command{
		Use: "init",
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestPath := filepath.Join(xdg.DataHome, "Google", "Chrome", "NativeMessagingHosts", "com.pomdtr.webterm.json")
			cmd.Printf("Writing manifest file to %s\n", manifestPath)
			if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
				return fmt.Errorf("unable to write manifest file: %w", err)
			}
			cmd.Printf("Manifest file written successfully\n")

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("unable to get user home directory: %w", err)
			}

			entrypointPath := filepath.Join(homeDir, ".local", "bin", "webterm.sh")
			cmd.Printf("Writing entrypoint file to %s\n", entrypointPath)
			if err := os.WriteFile(entrypointPath, []byte(entrypoint), 0755); err != nil {
				return fmt.Errorf("unable to write entrypoint file: %w", err)
			}
			cmd.Printf("Entrypoint file written successfully\n")

			return nil
		},
	}

	return cmd
}

func Execute() error {
	cmd := &cobra.Command{
		Use:          "webterm",
		SilenceUsage: true,
	}
	var isTTY bool
	var width int
	if isatty.IsTerminal(os.Stdout.Fd()) {
		isTTY = true
		w, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return err
		}
		width = w
	}

	printer := tableprinter.New(os.Stdout, isTTY, width)

	cmd.AddCommand(NewCmdInit())
	cmd.AddCommand(NewCmdServer())
	cmd.AddCommand(NewCmdTab(printer))
	cmd.AddCommand(NewCmdWindow(printer))
	cmd.AddCommand(NewCmdHistory())
	cmd.AddCommand(NewCmdExtension(printer))
	cmd.AddCommand(NewCmdBookMark())
	cmd.AddCommand(NewCmdDownload(printer))
	cmd.AddCommand(NewCmdSelection())

	return cmd.Execute()
}
