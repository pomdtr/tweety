package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	newTabUrl = "chrome://newtab/"
)

func sendMessage[T any](payload any) (T, error) {
	var data T

	env, ok := os.LookupEnv("POPCORN_PORT")
	if !ok {
		return data, fmt.Errorf("POPCORN_PORT is not set")
	}

	port, err := strconv.Atoi(env)
	if err != nil {
		return data, fmt.Errorf("POPCORN_PORT is not a number")
	}

	target := fmt.Sprintf("http://localhost:%d/browser", port)
	b, err := json.Marshal(payload)
	if err != nil {
		return data, err
	}

	res, err := http.Post(target, "application/json", bytes.NewReader(b))
	if err != nil {
		return data, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(res.Body)
		return data, fmt.Errorf(string(msg))
	}

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&data); err != nil {
		return data, err
	}

	return data, nil
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
	cmd.AddCommand(NewCmdServe())
	cmd.AddCommand(NewCmdTab(printer))
	cmd.AddCommand(NewCmdWindow(printer))
	cmd.AddCommand(NewCmdHistory())
	cmd.AddCommand(NewCmdExtension(printer))
	cmd.AddCommand(NewCmdBookMark())
	cmd.AddCommand(NewCmdDownload(printer))
	cmd.AddCommand(NewCmdProfile())

	return cmd.Execute()
}
