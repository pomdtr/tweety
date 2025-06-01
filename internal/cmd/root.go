package cmd

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/creack/pty"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	jsonparser "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/pomdtr/tweety/internal/jsonrpc"

	"github.com/spf13/cobra"
)

var k = koanf.New(".")

var (
	tweetyPort  int
	tweetyToken string
)

func init() {
	tweetyToken = os.Getenv("TWEETY_TOKEN")

	if tweetyPortStr := os.Getenv("TWEETY_PORT"); tweetyPortStr != "" {
		var err error
		tweetyPort, err = strconv.Atoi(tweetyPortStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid TWEETY_PORT environment variable: %v\n", err)
			os.Exit(1)
		}
	}
}

var (
	maxBufferSizeBytes   = 512
	keepalivePingTimeout = 20 * time.Second
)

type RequestParamsRunCommand struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Rows    uint16   `json:"rows"`
	Cols    uint16   `json:"cols"`
}

type RequestParamsResizeTTY struct {
	TTY  string `json:"tty"`
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

var configDir = filepath.Join(os.Getenv("HOME"), ".config", "tweety")
var cacheDir = filepath.Join(os.Getenv("HOME"), ".cache", "tweety")
var dataDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "tweety")

func NewCmdRoot() *cobra.Command {
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

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cmd := &cobra.Command{
		Use:          "tweety",
		SilenceUsage: true,
		Short:        "An integrated terminal for your web browser",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := k.Load(file.Provider(filepath.Join(configDir, "config.json")), jsonparser.Parser()); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			return nil
		},
	}

	cmd.AddCommand(
		NewCmdServe(),
		NewCmdInstall(),
		NewCmdUninstall(),
		NewCmdConfig(),
		NewCmdTabs(),
		NewCmdBookmarks(),
		NewCmdHistory(),
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
			port, err := getFreePort()
			if err != nil {
				return fmt.Errorf("failed to get free port: %w", err)
			}

			token := rand.Text()

			handler, err := NewHandler(HandlerParams{
				Port:  port,
				Token: token,
			})
			if err != nil {
				return err
			}

			log.Printf("Listening on http://localhost:%d\n", port)
			return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
		},
	}

	return cmd
}

//go:embed all:embed
var embedFs embed.FS

func NewCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Tweety native messaging host",
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
					"Path": hostPath,
				}); err != nil {
					return fmt.Errorf("failed to execute manifest template: %w", err)
				}
			}

			return nil
		},
	}

	return cmd
}

func NewCmdUninstall() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Tweety native messaging host",
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

func NewCmdConfig() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Open Tweety configuration file in your editor",
		RunE: func(cmd *cobra.Command, args []string) error {
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
	Port  int
	Token string
}

func NewHandler(handlerParams HandlerParams) (http.Handler, error) {
	var ttyMap = make(map[string]*os.File)
	host := jsonrpc.NewHost()

	host.HandleRequest("tty.create", func(input []byte) (any, error) {
		var requestParams RequestParamsRunCommand
		if err := json.Unmarshal(input, &requestParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal exec params: %w", err)
		}

		var command string
		var args []string
		if name := requestParams.Command; name != "" {
			commandsDir := filepath.Join(configDir, "commands")
			entries, err := os.ReadDir(commandsDir)

			if err != nil {
				return nil, fmt.Errorf("failed to read commands directory: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				entryName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				if entryName != name {
					continue
				}

				entryPath := filepath.Join(commandsDir, entry.Name())
				// make sure the command is executable
				if entry.Type().IsRegular() {
					if err := os.Chmod(entryPath, 0755); err != nil {
						return nil, fmt.Errorf("failed to make command executable: %w", err)
					}
				}

				command = filepath.Join(entryPath)
				args = requestParams.Args
				break
			}

			if command == "" {
				return nil, fmt.Errorf("command not found: %s", name)
			}

		} else {
			command = k.String("command")
			args = k.Strings("args")
		}

		cmd := exec.Command(command, args...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
		cmd.Env = append(cmd.Env, fmt.Sprintf("TWEETY_PORT=%d", handlerParams.Port))
		cmd.Env = append(cmd.Env, fmt.Sprintf("TWEETY_TOKEN=%s", handlerParams.Token))
		for key, value := range k.StringMap("env") {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}

		cmd.Dir = os.Getenv("HOME")
		log.Println("executing command:", cmd.String())
		tty, err := pty.Start(cmd)
		if err != nil {
			log.Printf("failed to start pty: %s", err)
			return nil, fmt.Errorf("failed to start pty: %w", err)
		}

		if err := pty.Setsize(tty, &pty.Winsize{Rows: requestParams.Rows, Cols: requestParams.Cols}); err != nil {
			log.Printf("failed to set size for tty: %s", err)
			return nil, fmt.Errorf("failed to set size for tty: %w", err)
		}

		ttyID := strings.ToLower(rand.Text())
		ttyMap[ttyID] = tty

		return map[string]string{
			"url": fmt.Sprintf("ws://127.0.0.1:%d/tty/%s", handlerParams.Port, ttyID),
			"id":  ttyID,
		}, nil

	})

	host.HandleNotification("tty.resize", func(input []byte) error {
		var requestParams RequestParamsResizeTTY
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

	host.HandleRequest("config.get", func(input []byte) (any, error) {
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

	go host.Listen()

	r := chi.NewRouter()

	r.Post("/jsonrpc", func(w http.ResponseWriter, r *http.Request) {
		// check bearer token
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != handlerParams.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var request jsonrpc.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("failed to decode request: %s", err), http.StatusBadRequest)
			return
		}

		resp, err := host.Send(request)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to send request: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, fmt.Sprintf("failed to encode response: %s", err), http.StatusInternalServerError)
			return
		}
	})

	r.Get("/tty/{id}", func(w http.ResponseWriter, r *http.Request) {
		ttyID := chi.URLParam(r, "id")

		tty, ok := ttyMap[ttyID]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid terminal ID: %s", ttyID)))
			return
		}

		defer func() {
			// cleanup
			delete(ttyMap, ttyID)
			// send the signal to the process
			tty.Close()
		}()

		HandleWebsocket(tty)(w, r)
	})

	return r, nil
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
