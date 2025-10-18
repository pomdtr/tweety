package cmd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/pomdtr/tweety/internal/jsonrpc"
	"github.com/spf13/cobra"
)

//go:embed all:themes
var themeFs embed.FS

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

			ttyMap := make(map[string]*os.File)
			messagingHost := NewMessagingHost(logger, port, ttyMap)

			handler := NewWebSocketHandler(ttyMap)

			logger.Info("Listening", "port", port)

			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", port),
				Handler: handler,
			}

			// Channel to signal when messaging host stops
			done := make(chan error, 1)

			// Start messaging host
			go func() {
				if err := messagingHost.Listen(); err != nil {
					logger.Error("Messaging host listen loop exited", "error", err)
					done <- err
				} else {
					logger.Info("Messaging host stopped normally")
					done <- nil
				}
			}()

			// Start HTTP server
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP server error", "error", err)
					done <- err
				}
			}()

			// Wait for either messaging host to stop or server error
			err = <-done
			logger.Info("Shutting down server")
			server.Shutdown(context.Background())
			return err
		},
	}

	return cmd
}

type HandlerParams struct {
	Logger *slog.Logger
	Port   int
}

type MessagingHostParams struct {
	Logger *slog.Logger
	Port   int
	TTYMap map[string]*os.File
}

type Command struct {
	ID   string          `json:"id"`
	Meta CommandMetadata `json:"meta"`
}

// CommandMetadata holds the structured metadata extracted from the script file.
type CommandMetadata struct {
	Title               string   `json:"title"`
	Contexts            []string `json:"contexts"`
	DocumentUrlPatterns []string `json:"documentUrlPatterns,omitempty"`
	TargetUrlPatterns   []string `json:"targetUrlPatterns,omitempty"`
}

func NewMessagingHost(logger *slog.Logger, port int, ttyMap map[string]*os.File) *jsonrpc.Host {
	messagingHost := jsonrpc.NewHost(logger)

	messagingHost.HandleRequest("initialize", func(input []byte) (any, error) {
		var params struct {
			Version   string `json:"version"`
			BrowserID string `json:"browserId"`
		}

		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal initialize params: %w", err)
		}

		logger.Info("Received initialize notification", "version", params.Version, "browserId", params.BrowserID)
		socketPath := filepath.Join(cacheDir, "sockets", fmt.Sprintf("%s.sock", params.BrowserID))
		if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create socket directory: %w", err)
		}

		os.Setenv("TWEETY_SOCKET", socketPath)
		if _, err := os.Stat(socketPath); err == nil {
			if err := os.Remove(socketPath); err != nil {
				log.Printf("Failed to remove existing socket file: %s", err)
				return nil, fmt.Errorf("failed to remove existing socket file: %w", err)
			}
		}

		// create a listener
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			logger.Error("Failed to create unix socket listener", "error", err)
			return nil, fmt.Errorf("failed to create unix socket listener: %w", err)
		}

		// scanCommands reads the commandDir and sends a commands.update notification.
		scanCommands := func() error {
			entries, err := os.ReadDir(commandDir)
			if err != nil {
				return fmt.Errorf("failed to read command directory: %w", err)
			}

			var commands []Command
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				f, err := os.Open(filepath.Join(commandDir, entry.Name()))
				if err != nil {
					logger.Error("Failed to open command file", "file", entry.Name(), "error", err)
					continue
				}
				// ensure file is closed after processing this file
				func() {
					defer f.Close()

					metadata, err := ExtractMetadata(f)
					if err != nil {
						logger.Error("Failed to extract metadata from command file", "file", entry.Name(), "error", err)
						return
					}

					if metadata.Title == "" {
						return
					}

					if len(metadata.Contexts) == 0 {
						return
					}

					commands = append(commands, Command{
						ID:   entry.Name(),
						Meta: metadata,
					})
				}()
			}

			if err := messagingHost.SendNotification("commands.update", [][]Command{commands}); err != nil {
				logger.Error("Failed to send commands.update notification", "error", err)
				return fmt.Errorf("failed to send commands.update notification: %w", err)
			}

			return nil
		}

		// Run an initial scan immediately
		logger.Info("Running initial command scan")
		if err := scanCommands(); err != nil {
			logger.Error("initial scan failed", "error", err)
			// Do not return an error here; allow initialize to continue but report back
		}

		// Start fsnotify watcher on appDir to re-run the scan whenever the appDir changes.
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			logger.Error("failed to create fsnotify watcher", "error", err)
		} else {
			// attempt to add appDir to watcher
			if err := watcher.Add(commandDir); err != nil {
				logger.Error("failed to add appDir to watcher", "appDir", appDir, "error", err)
			} else {
				// debounced scanning to coalesce bursts of file events
				var mu sync.Mutex
				var pending bool

				scheduleScan := func() {
					mu.Lock()
					if pending {
						mu.Unlock()
						return
					}
					pending = true
					mu.Unlock()

					go func() {
						time.Sleep(150 * time.Millisecond)
						mu.Lock()
						pending = false
						mu.Unlock()
						if err := scanCommands(); err != nil {
							logger.Error("scanCommands failed after fsnotify event", "error", err)
						}
					}()
				}

				go func() {
					for {
						select {
						case ev, ok := <-watcher.Events:
							if !ok {
								return
							}
							// react to write/create/remove/rename events
							if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod) != 0 {
								logger.Info("fsnotify event detected, rescanning commands")
								scheduleScan()
							}
						case err, ok := <-watcher.Errors:
							if !ok {
								return
							}
							logger.Error("fsnotify watcher error", "error", err)
						}
					}
				}()
			}
		}

		go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var request jsonrpc.JSONRPCRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, fmt.Sprintf("failed to decode request: %s", err), http.StatusBadRequest)
				return
			}

			resp, err := messagingHost.SendRequest(request)
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

		return map[string]any{}, nil
	})

	messagingHost.HandleRequest("commands.run", func(input []byte) (any, error) {
		var params struct {
			Command string          `json:"command"`
			Input   json.RawMessage `json:"input"`
		}

		logger.Info("Received commands.run notification", "input", string(params.Input))

		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command params: %w", err)
		}

		entrypoint := filepath.Join(commandDir, params.Command)
		if _, err := os.Stat(entrypoint); err != nil {
			return nil, fmt.Errorf("failed to stat command entrypoint: %w", err)
		}

		cmd := exec.Command(entrypoint)
		cmd.Stdin = bytes.NewReader(params.Input)
		cmd.Env = os.Environ()
		for key, value := range k.StringMap("env") {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}

		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if ok := errors.As(err, &exitErr); ok {
				// Command exited with non-zero status
				return nil, fmt.Errorf("command exited with status %d, stderr: %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
			return nil, fmt.Errorf("failed to run command: %w", err)
		}

		return nil, nil
	})

	messagingHost.HandleRequest("tty.create", func(input []byte) (any, error) {
		var params struct {
			Mode string   `json:"mode"`
			App  string   `json:"app"`
			Args []string `json:"args"`
			Cwd  string   `json:"cwd"`
			File string   `json:"file"`
		}

		if len(input) > 0 {
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, fmt.Errorf("failed to unmarshal create params: %w", err)
			}
		}

		var cmd *exec.Cmd

		if params.Mode == "app" && params.App != "" {
			// First try to find the exact file name
			entrypoint := filepath.Join(appDir, params.App)
			stat, err := os.Stat(entrypoint)

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

						if nameWithoutExt == params.App {
							entrypoint = filepath.Join(appDir, name)
							stat, err = os.Stat(entrypoint)
							break
						}
					}
				}
			}

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

			cmd = exec.Command(entrypoint, params.Args...)
		} else if params.Mode == "editor" && params.File != "" {
			editor := k.String("editor")
			if editor == "" {
				editor = getDefaultEditor()
			}

			cmd = exec.Command(editor, params.File)
		} else {
			cmd = exec.Command(k.String("command"), k.Strings("args")...)
		}

		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
		cmd.Env = append(cmd.Env, "TERM_PROGRAM=tweety")
		for key, value := range k.StringMap("env") {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}

		if params.Cwd != "" {
			cmd.Dir = params.Cwd
		} else {
			cmd.Dir = os.Getenv("HOME")
		}

		log.Println("executing command:", cmd.String())
		tty, err := pty.Start(cmd)
		if err != nil {
			log.Printf("failed to start pty: %s", err)
			return nil, fmt.Errorf("failed to start pty: %w", err)
		}

		ttyID := strings.ToLower(rand.Text())
		ttyMap[ttyID] = tty

		return map[string]string{
			"url": fmt.Sprintf("ws://127.0.0.1:%d/tty/%s", port, ttyID),
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
		var params struct {
			Variant string `json:"variant"`
		}

		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal xterm config params: %w", err)
		}

		var theme string
		if darkTheme := k.String("themeDark"); params.Variant == "dark" && darkTheme != "" {
			theme = darkTheme
		} else {
			theme = k.String("theme")
		}

		themeBytes, err := themeFs.ReadFile(filepath.Join("themes", theme+".json"))
		if err != nil {
			return nil, fmt.Errorf("failed to read theme file: %w", err)
		}

		xtermConfig := map[string]interface{}{
			"cursorBlink":                   true,
			"allowProposedApi":              true,
			"macOptionIsMeta":               true,
			"macOptionClickForcesSelection": true,
			"fontSize":                      13,
			"fontFamily":                    "Consolas,Liberation Mono,Menlo,Courier,monospace",
			"theme":                         json.RawMessage(themeBytes),
		}

		if err := k.Unmarshal("xterm", &xtermConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal xterm config: %w", err)
		}

		return xtermConfig, nil
	})

	messagingHost.HandleRequest("readFile", func(input []byte) (any, error) {
		var params struct {
			Path string `json:"path"`
		}

		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal readFile params: %w", err)
		}

		content, err := os.ReadFile(params.Path)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			return map[string]any{
				"content": "",
			}, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		return map[string]any{
			"content": string(content),
		}, nil
	})

	messagingHost.HandleRequest("writeFile", func(input []byte) (any, error) {
		var params struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}

		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal writeFile params: %w", err)
		}

		f, err := os.OpenFile(params.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open file for writing: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(params.Content); err != nil {
			return nil, fmt.Errorf("failed to write file content: %w", err)
		}

		return map[string]any{}, nil
	})

	return messagingHost
}

func NewWebSocketHandler(ttyMap map[string]*os.File) http.Handler {
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

func ExtractMetadata(reader io.Reader) (CommandMetadata, error) {
	var result CommandMetadata
	scanner := bufio.NewScanner(reader)

	metadataRegex := regexp.MustCompile(`^.*?@tweety\.(\w+)\s+(.*)\s*$`)

	for scanner.Scan() {
		line := scanner.Text() // Get the raw line

		matches := metadataRegex.FindStringSubmatch(line)
		if len(matches) < 3 {
			// No match, or not enough capturing groups (expecting 3: full match, key, value)
			continue
		}

		key := matches[1]      // The captured key (e.g., "title")
		rawValue := matches[2] // The captured value (e.g., "Open in archive.ph" or `["all"]`)

		switch key {
		case "title":
			result.Title = rawValue
		case "contexts":
			var contextsRaw []string
			err := json.Unmarshal([]byte(rawValue), &contextsRaw)
			if err != nil {
				return CommandMetadata{}, fmt.Errorf("failed to parse 'contexts' as string array from '%s': %w", rawValue, err)
			}
			result.Contexts = contextsRaw
		case "documentUrlPatterns":
			var patternsRaw []string
			if err := json.Unmarshal([]byte(rawValue), &patternsRaw); err != nil {
				return CommandMetadata{}, fmt.Errorf("failed to parse 'documentUrlPatterns' as string array from '%s': %w", rawValue, err)
			}
			result.DocumentUrlPatterns = patternsRaw
		case "targetUrlPatterns":
			var patternsRaw []string
			if err := json.Unmarshal([]byte(rawValue), &patternsRaw); err != nil {
				return CommandMetadata{}, fmt.Errorf("failed to parse 'targetUrlPatterns' as string array from '%s': %w", rawValue, err)
			}
			result.TargetUrlPatterns = patternsRaw
		}
	}

	if err := scanner.Err(); err != nil {
		return CommandMetadata{}, err
	}

	return result, nil
}
