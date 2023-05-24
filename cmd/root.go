package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const popcornPort = 9999

const (
	newTabUrl = "chrome://newtab/"
)

func sendMessage(payload any) ([]byte, error) {
	target := fmt.Sprintf("http://localhost:%d/browser", popcornPort)
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

func Execute() error {
	cmd := &cobra.Command{
		Use:          "popcorn",
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
