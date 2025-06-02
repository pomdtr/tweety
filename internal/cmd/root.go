package cmd

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	jsonparser "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"

	"github.com/knadh/koanf/v2"
	"github.com/pomdtr/tweety/internal/jsonrpc"

	"github.com/spf13/cobra"
)

var k = koanf.New(".")

var (
	maxBufferSizeBytes   = 512
	keepalivePingTimeout = 20 * time.Second
)

var configDir = filepath.Join(os.Getenv("HOME"), ".config", "tweety")
var cacheDir = filepath.Join(os.Getenv("HOME"), ".cache", "tweety")
var dataDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "tweety")

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "tweety",
		SilenceUsage: true,
		Short:        "An integrated terminal for your web browser",
		Args:         cobra.ExactArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			confmapProvider := confmap.Provider(map[string]interface{}{
				"command": getDefaultShell(),
			}, ".")
			if err := k.Load(confmapProvider, nil); err != nil {
				return fmt.Errorf("failed to load default config: %w", err)
			}

			f := file.Provider(filepath.Join(configDir, "config.json"))
			if err := k.Load(f, jsonparser.Parser()); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			f.Watch(func(event interface{}, err error) {
				if err != nil {
					log.Printf("watch error: %v", err)
					return
				}

				k = koanf.New(".")
				k.Load(f, jsonparser.Parser())
			})

			return nil
		},
	}

	cmd.AddCommand(
		NewCmdServe(),
		NewCmdInstall(),
		NewCmdUninstall(),
		NewCmdTabs(),
		NewCmdBookmarks(),
		NewCmdHistory(),
		NewCmdWindows(),
		NewCmdNotifications(),
		NewCmdEdit(),
		NewCmdSSH(),
		NewCmdOpen(),
		NewCmdConfig(),
		NewCmdApps(),
	)

	return cmd
}

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Hidden:       true,
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
				os.Exit(1)
			}

			logFile, err := os.OpenFile(filepath.Join(cacheDir, "log.txt"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
				os.Exit(1)
			}
			defer logFile.Close()
			// create new slog logger
			logger := slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{}))

			port, err := getFreePort()
			if err != nil {
				return fmt.Errorf("failed to get free port: %w", err)
			}

			handler := NewHandler(HandlerParams{
				Logger: logger,
				Port:   port,
			})

			logger.Info("Listening", "port", port)
			return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
		},
	}

	return cmd
}

//go:embed all:embed
var embedFs embed.FS

func NewCmdInstall() *cobra.Command {
	var flags struct {
		extensionID string
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install native messaging host",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %w", err)
			}

			hostTemplate, err := template.ParseFS(embedFs, "embed/native_messaging_host.tmpl")
			if err != nil {
				return fmt.Errorf("failed to parse template: %w", err)
			}

			hostPath := filepath.Join(dataDir, "native_messaging_host")
			f, err := os.Create(hostPath)
			if err != nil {
				return fmt.Errorf("failed to create native messaging host file: %w", err)
			}
			defer f.Close()

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}

			if err := hostTemplate.Execute(f, map[string]interface{}{
				"ExecPath": execPath,
			}); err != nil {
				return fmt.Errorf("failed to execute template: %w", err)
			}

			if err := os.Chmod(hostPath, 0755); err != nil {
				return fmt.Errorf("failed to make host file executable: %w", err)
			}

			manifestTemplate, err := template.ParseFS(embedFs, "embed/com.github.pomdtr.tweety.json.tmpl")
			if err != nil {
				return fmt.Errorf("failed to parse manifest template: %w", err)
			}

			dirs, err := getManifestDirs()
			if err != nil {
				return fmt.Errorf("failed to get manifest directories: %w", err)
			}

			for _, dir := range dirs {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					continue
				}

				f, err := os.Create(filepath.Join(dir, "com.github.pomdtr.tweety.json"))
				if err != nil {
					return fmt.Errorf("failed to get manifest file path: %w", err)
				}
				defer f.Close()

				if err := manifestTemplate.Execute(f, map[string]interface{}{
					"Path":        hostPath,
					"ExtensionID": flags.extensionID,
				}); err != nil {
					return fmt.Errorf("failed to execute manifest template: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.extensionID, "extension-id", "", "Extension ID for the native messaging host")
	cmd.MarkFlagRequired("extension-id")

	return cmd
}

func NewCmdUninstall() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall native messaging host",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := getManifestDirs()
			if err != nil {
				return fmt.Errorf("failed to get manifest directories: %w", err)
			}

			for _, dir := range dirs {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					continue
				}

				manifestPath := filepath.Join(dir, "com.github.pomdtr.tweety.json")
				if err := os.Remove(manifestPath); err != nil {
					return fmt.Errorf("failed to remove manifest file: %w", err)
				}
			}

			hostPath := filepath.Join(dataDir, "native_messaging_host")
			if err := os.Remove(hostPath); err != nil {
				return fmt.Errorf("failed to remove native messaging host file: %w", err)
			}

			return nil
		},
	}
}

func getManifestDirs() ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		supportDir := filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
		return []string{
			filepath.Join(supportDir, "Google", "Chrome", "NativeMessagingHosts"),
			filepath.Join(supportDir, "Chromium", "NativeMessagingHosts"),
			filepath.Join(supportDir, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"),
			filepath.Join(supportDir, "Vivaldi", "NativeMessagingHosts"),
			filepath.Join(supportDir, "Microsoft", "Edge", "NativeMessagingHosts"),
			filepath.Join(supportDir, "Firefox", "NativeMessagingHosts"),
			filepath.Join(supportDir, "Zen", "NativeMessagingHosts"),
		}, nil
	case "linux":
		configDir := filepath.Join(os.Getenv("HOME"), ".config")
		return []string{
			filepath.Join(os.Getenv("HOME"), ".mozilla", "native-messaging-hosts"),
			filepath.Join(configDir, "google-chrome", "NativeMessagingHosts"),
			filepath.Join(configDir, "chromium", "NativeMessagingHosts"),
			filepath.Join(configDir, "microsoft-edge", "NativeMessagingHosts"),
		}, nil
	}

	return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
}

type HandlerParams struct {
	Logger *slog.Logger
	Port   int
}

type ScriptCommand struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

func NewHandler(handlerParams HandlerParams) http.Handler {
	var ttyMap = make(map[string]*os.File)
	messagingHost := jsonrpc.NewHost(handlerParams.Logger)

	messagingHost.HandleNotification("initialize", func(input []byte) error {
		var params struct {
			Version   string `json:"version"`
			BrowserID string `json:"browserId"`
		}

		if err := json.Unmarshal(input, &params); err != nil {
			return fmt.Errorf("failed to unmarshal initialize params: %w", err)
		}

		handlerParams.Logger.Info("Received initialize notification", "version", params.Version, "browserId", params.BrowserID)
		socketPath := filepath.Join(cacheDir, "sockets", fmt.Sprintf("%s.sock", params.BrowserID))
		if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
			return fmt.Errorf("failed to create socket directory: %w", err)
		}

		os.Setenv("TWEETY_SOCKET", socketPath)
		if _, err := os.Stat(socketPath); err == nil {
			if err := os.Remove(socketPath); err != nil {
				return fmt.Errorf("failed to remove existing socket file: %w", err)
			}
		}

		// create a listener
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			return fmt.Errorf("failed to create unix socket listener: %w", err)
		}

		return http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var request jsonrpc.JSONRPCRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, fmt.Sprintf("failed to decode request: %s", err), http.StatusBadRequest)
				return
			}

			resp, err := messagingHost.Send(request)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to send request: %s", err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, fmt.Sprintf("failed to encode response: %s", err), http.StatusInternalServerError)
				return
			}
		}))
	})

	messagingHost.HandleRequest("tty.create", func(input []byte) (any, error) {
		var createParams struct {
			Mode string `json:"mode"`
			File string `json:"file"`
			Host string `json:"host"`
			App  string `json:"app"`
		}

		if err := json.Unmarshal(input, &createParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal create params: %w", err)
		}

		var cmd *exec.Cmd
		if createParams.Mode == "terminal" {
			var args []string
			if command := k.String("command"); strings.Contains(command, " ") {
				args = []string{"/bin/sh", "-c", command}
			} else {
				args = []string{command}
			}
			cmd = exec.Command(args[0], args[1:]...)
		} else if createParams.Mode == "ssh" {
			if createParams.Host == "" {
				return nil, fmt.Errorf("host is required for ssh mode")
			}

			cmd = exec.Command("ssh", createParams.Host)
		} else if createParams.Mode == "editor" {
			if createParams.File == "" {
				return nil, fmt.Errorf("file is required for editor mode")
			}

			if _, err := os.Stat(createParams.File); os.IsNotExist(err) {
				return nil, fmt.Errorf("file does not exist: %s", createParams.File)
			}

			editor := k.String("editor")
			if editor == "" {
				if editorEnv := os.Getenv("EDITOR"); editorEnv != "" {
					editor = editorEnv
				} else {
					editor = "vi" // default editor
				}
			}

			cmd = exec.Command("sh", "-c", fmt.Sprintf("%s %s", editor, createParams.File))
		} else if createParams.Mode == "app" {
			entrypoint := filepath.Join(appDir, createParams.App)
			stat, err := os.Stat(entrypoint)
			if err != nil {
				return nil, fmt.Errorf("failed to stat app entrypoint: %w", err)
			}

			if stat.IsDir() {
				return nil, fmt.Errorf("app entrypoint is a directory, expected a file: %s", entrypoint)
			}

			// check if the entrypoint is executable
			if stat.Mode()&0111 == 0 {
				if err := os.Chmod(entrypoint, 0755); err != nil {
					return nil, fmt.Errorf("failed to make app entrypoint executable: %w", err)
				}
			}

			cmd = exec.Command(entrypoint)
		} else {
			return nil, fmt.Errorf("invalid mode: %s", createParams.Mode)
		}

		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
		cmd.Env = append(cmd.Env, "TERM_PROGRAM=tweety")

		cmd.Dir = os.Getenv("HOME")
		log.Println("executing command:", cmd.String())
		tty, err := pty.Start(cmd)
		if err != nil {
			log.Printf("failed to start pty: %s", err)
			return nil, fmt.Errorf("failed to start pty: %w", err)
		}

		ttyID := strings.ToLower(rand.Text())
		ttyMap[ttyID] = tty

		return map[string]string{
			"url": fmt.Sprintf("ws://127.0.0.1:%d/tty/%s", handlerParams.Port, ttyID),
			"id":  ttyID,
		}, nil

	})

	messagingHost.HandleNotification("tty.resize", func(input []byte) error {
		var requestParams struct {
			TTY  string `json:"tty"`
			Rows uint16 `json:"rows"`
			Cols uint16 `json:"cols"`
		}
		if err := json.Unmarshal(input, &requestParams); err != nil {
			return fmt.Errorf("failed to unmarshal resize params: %w", err)
		}

		tty, ok := ttyMap[requestParams.TTY]
		if !ok {
			return fmt.Errorf("invalid tty ID: %s", requestParams.TTY)
		}

		if err := pty.Setsize(tty, &pty.Winsize{Rows: requestParams.Rows, Cols: requestParams.Cols}); err != nil {
			return fmt.Errorf("failed to set size for tty: %w", err)
		}

		return nil
	})

	messagingHost.HandleRequest("xterm.getConfig", func(input []byte) (any, error) {
		xtermConfig := map[string]interface{}{
			"cursorBlink":                   true,
			"allowProposedApi":              true,
			"macOptionIsMeta":               true,
			"macOptionClickForcesSelection": true,
			"fontSize":                      13,
			"fontFamily":                    "Consolas,Liberation Mono,Menlo,Courier,monospace",
			"theme": map[string]interface{}{
				"foreground":          "#c5c8c6",
				"background":          "#1d1f21",
				"ansiBlack":           "#000000",
				"ansiBlue":            "#81a2be",
				"ansiCyan":            "#8abeb7",
				"ansiGreen":           "#b5bd68",
				"ansiMagenta":         "#b294bb",
				"ansiRed":             "#cc6666",
				"ansiWhite":           "#ffffff",
				"ansiYellow":          "#f0c674",
				"ansiBrightBlack":     "#000000",
				"ansiBrightBlue":      "#81a2be",
				"ansiBrightCyan":      "#8abeb7",
				"ansiBrightGreen":     "#b5bd68",
				"ansiBrightMagenta":   "#b294bb",
				"ansiBrightRed":       "#cc6666",
				"ansiBrightWhite":     "#ffffff",
				"ansiBrightYellow":    "#f0c674",
				"selectionBackground": "#373b41",
				"cursor":              "#c5c8c6",
			},
		}

		if err := k.Unmarshal("xterm", &xtermConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal xterm config: %w", err)
		}

		return xtermConfig, nil
	})

	go messagingHost.Listen()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ttyID := strings.TrimPrefix(r.URL.Path, "/tty/")
		tty, ok := ttyMap[ttyID]
		if !ok {
			http.Error(w, fmt.Sprintf("invalid terminal ID: %s", ttyID), http.StatusBadRequest)
			return
		}

		defer func() {
			delete(ttyMap, ttyID)
			tty.Close()
		}()

		HandleWebsocket(tty)(w, r)
	})
}

func HandleWebsocket(tty *os.File) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Print("established connection identity")
		upgrader := getConnectionUpgrader(maxBufferSizeBytes)
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("failed to upgrade connection: %s", err)
			return
		}
		defer connection.Close()

		var waiter sync.WaitGroup
		waiter.Add(1)

		// tty << xterm.js
		go func() {
			for {
				// data processing
				_, data, err := connection.ReadMessage()
				if err != nil {
					log.Printf("failed to get next reader: %s", err)
					waiter.Done()
					return
				}
				dataLength := len(data)
				dataBuffer := bytes.Trim(data, "\x00")

				// process
				if dataLength == -1 { // invalid
					log.Printf("failed to get the correct number of bytes read, ignoring message")
					continue
				}

				// write to tty
				if _, err := tty.Write(dataBuffer); err != nil {
					log.Printf("failed to write %v bytes to tty: %s", len(dataBuffer), err)
					continue
				}
			}
		}()

		messages := make(chan []byte)
		// tty >> xterm.js
		go func() {
			for {
				buffer := make([]byte, maxBufferSizeBytes)
				readLength, err := tty.Read(buffer)
				if err != nil {
					connection.Close()
					log.Printf("failed to read from tty: %s", err)
					return
				}

				messages <- buffer[:readLength]
			}
		}()

		lastPingTime := time.Now()
		connection.SetPongHandler(func(appData string) error {
			lastPingTime = time.Now()
			return nil
		})

		// this is a keep-alive loop that ensures connection does not hang-up itself
		go func() {
			ticker := time.NewTicker(keepalivePingTimeout / 2)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := connection.WriteMessage(websocket.PingMessage, []byte("keepalive")); err != nil {
						log.Printf("failed to write ping message")
					}

					if time.Since(lastPingTime) > keepalivePingTimeout {
						log.Printf("connection timeout, closing connection")
						connection.Close()
						return
					}
				case m := <-messages:
					if err := connection.WriteMessage(websocket.BinaryMessage, m); err != nil {
						log.Printf("failed to send %v bytes from tty to xterm.js", len(m))
						continue
					}
				}
			}
		}()

		waiter.Wait()
	}
}

func getConnectionUpgrader(
	maxBufferSizeBytes int,
) websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		HandshakeTimeout: 0,
		ReadBufferSize:   maxBufferSizeBytes,
		WriteBufferSize:  maxBufferSizeBytes,
	}
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func getDefaultShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		switch runtime.GOOS {
		case "darwin":
			return "/bin/zsh"
		case "linux":
			return "/bin/bash"
		default:
			return "/bin/sh"
		}
	}
	return shell
}
