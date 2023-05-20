package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/adrg/xdg"
	"github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/joho/godotenv"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func sendMessage(payload any) ([]byte, error) {
	target := fmt.Sprintf("http://localhost:%d/browser", webTermPort)
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	res, err := http.Post(target, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	return io.ReadAll(res.Body)
}

func getPrinter() (tableprinter.TablePrinter, error) {
	var isTTY bool
	var width int
	if isatty.IsTerminal(os.Stdout.Fd()) {
		isTTY = true
		w, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return nil, err
		}
		width = w
	}

	return tableprinter.New(os.Stdout, isTTY, width), nil
}

type Tab struct {
	Active          bool   `json:"active"`
	Audible         bool   `json:"audible"`
	AutoDiscardable bool   `json:"autoDiscardable"`
	Discarded       bool   `json:"discarded"`
	FavIconURL      string `json:"favIconUrl"`
	GroupID         int    `json:"groupId"`
	Height          int    `json:"height"`
	Highlighted     bool   `json:"highlighted"`
	ID              int    `json:"id"`
	Incognito       bool   `json:"incognito"`
	Index           int    `json:"index"`
	MutedInfo       struct {
		Muted bool `json:"muted"`
	} `json:"mutedInfo"`
	Pinned   bool   `json:"pinned"`
	Selected bool   `json:"selected"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	WindowID int    `json:"windowId"`
}

func NewCmdTabList() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]string{
				"command": "tab.list",
			})
			if err != nil {
				return err
			}

			var tabs []Tab
			if err := json.Unmarshal(res, &tabs); err != nil {
				return err
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(tabs); err != nil {
					return err
				}
				return nil
			}

			printer, err := getPrinter()
			if err != nil {
				return err
			}

			for _, tab := range tabs {
				printer.AddField(tab.Title)
				printer.AddField(tab.URL)
				printer.EndRow()
			}

			if err := printer.Render(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "output as json")

	return cmd
}

func NewCmdTabCreate() *cobra.Command {
	return &cobra.Command{
		Use:  "create",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			_, err := sendMessage(map[string]any{
				"command": "tab.create",
				"url":     url,
			})

			if err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdTabFocus() *cobra.Command {
	return &cobra.Command{
		Use:  "focus",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabId, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}

			if _, err := sendMessage(map[string]any{
				"command": "tab.focus",
				"tabId":   tabId,
			}); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdTabClose() *cobra.Command {
	return &cobra.Command{
		Use:  "close",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabId, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}

			if _, err := sendMessage(map[string]any{
				"command": "tab.remove",
				"tabId":   tabId,
			}); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdWindowList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]string{
				"command": "window.list",
			})
			if err != nil {
				return err
			}

			if _, err := os.Stdout.Write(res); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdHistorySearch() *cobra.Command {
	return &cobra.Command{
		Use:  "search",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			res, err := sendMessage(map[string]any{
				"command": "history.search",
				"query":   query,
			})
			if err != nil {
				return err
			}

			if _, err := os.Stdout.Write(res); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdBookmarkList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]string{
				"command": "bookmark.list",
			})
			if err != nil {
				return err
			}

			if _, err := os.Stdout.Write(res); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdDownloadList() *cobra.Command {
	return &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := sendMessage(map[string]string{
				"command": "download.list",
			})
			if err != nil {
				return err
			}

			if _, err := os.Stdout.Write(res); err != nil {
				return err
			}

			return nil
		},
	}
}

func NewCmdWindow() *cobra.Command {
	cmd := &cobra.Command{
		Use: "window",
	}

	cmd.AddCommand(NewCmdWindowList())

	return cmd
}

func NewCmdBookMark() *cobra.Command {
	cmd := &cobra.Command{
		Use: "bookmark",
	}

	cmd.AddCommand(NewCmdBookmarkList())

	return cmd
}

func NewCmdTab() *cobra.Command {
	cmd := &cobra.Command{
		Use: "tab",
	}

	cmd.AddCommand(NewCmdTabList())
	cmd.AddCommand(NewCmdTabFocus())
	cmd.AddCommand(NewCmdTabCreate())

	return cmd
}

func NewCmdHistory() *cobra.Command {
	cmd := &cobra.Command{
		Use: "history",
	}

	cmd.AddCommand(NewCmdHistorySearch())

	return cmd
}

func NewCmdDownload() *cobra.Command {
	cmd := &cobra.Command{
		Use: "download",
	}

	cmd.AddCommand(NewCmdDownloadList())

	return cmd
}

func NewCmdServer() *cobra.Command {
	return &cobra.Command{
		Use:    "server",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			envFile := filepath.Join(xdg.ConfigHome, "webterm", "webterm.env")
			environ := os.Environ()
			environ = append(environ, "TERM=xterm-256color")

			if _, err := os.Stat(envFile); err == nil {
				env, err := godotenv.Read(envFile)
				if err != nil {
					return err
				}

				for k, v := range env {
					environ = append(environ, fmt.Sprintf("%s=%s", k, v))
				}
			}

			messageHandler := NewMessageHandler()
			server := NewServer(messageHandler, environ)

			go messageHandler.loop()
			if err := server.ListenAndServe(); err != nil {
				return err
			}

			return nil
		},
	}
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

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "webterm",
	}

	cmd.AddCommand(NewCmdInit())
	cmd.AddCommand(NewCmdServer())
	cmd.AddCommand(NewCmdTab())
	cmd.AddCommand(NewCmdWindow())
	cmd.AddCommand(NewCmdHistory())
	cmd.AddCommand(NewCmdBookMark())
	cmd.AddCommand(NewCmdDownload())
	return cmd
}
