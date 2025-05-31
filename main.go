package main

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	jsonparser "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"

	"github.com/spf13/cobra"
)

var k = koanf.New(".")

var (
	maxBufferSizeBytes   = 512
	keepalivePingTimeout = 20 * time.Second
)

type JSONRPCRequest struct {
	JSONRPCVersion string          `json:"jsonrpc"`
	ID             string          `json:"id"`
	Method         string          `json:"method"`
	Params         json.RawMessage `json:"params"`
}

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

func main() {
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
		Use:   "tweety",
		Short: "An integrated terminal for your web browser",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := k.Load(file.Provider(filepath.Join(configDir, "config.json")), jsonparser.Parser()); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			return nil
		},
	}

	cmd.AddCommand(NewCmdServe())
	cmd.AddCommand(NewCmdInstall())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
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
	r := chi.NewRouter()

	var ttyMap = make(map[string]*os.File)
	host := NewHost()

	host.HandleRequest("create_tty", func(input []byte) (any, error) {
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

	host.HandleNotification("resize_tty", func(input []byte) error {
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

	host.HandleRequest("get_xterm_config", func(input []byte) (any, error) {
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

	// Middleware to set the required header for private network access
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "same-origin")
			w.Header().Set("Content-Security-Policy", "script-src 'self';")
			next.ServeHTTP(w, r)
		})
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

type RequestHandlerFunc = func(params []byte) (result any, err error)
type NotificationHandlerFunc = func(params []byte) error

type Host struct {
	mu                   sync.Mutex
	requestsHandler      map[string]RequestHandlerFunc
	notificationsHandler map[string]NotificationHandlerFunc
	clientChannels       map[string]chan JSONRPCResponse
}

func (h *Host) HandleRequest(method string, handler RequestHandlerFunc) {
	h.requestsHandler[method] = handler
}

func (h *Host) HandleNotification(method string, handler NotificationHandlerFunc) {
	h.notificationsHandler[method] = handler
}

func NewHost() *Host {
	return &Host{
		requestsHandler:      make(map[string]RequestHandlerFunc),
		notificationsHandler: make(map[string]NotificationHandlerFunc),
		clientChannels:       make(map[string]chan JSONRPCResponse),
	}
}

func (h *Host) Listen() {
	for {
		lengthBytes := make([]byte, 4)
		if _, err := io.ReadFull(os.Stdin, lengthBytes); err != nil {
			continue
		}

		length := binary.LittleEndian.Uint32(lengthBytes)

		msgBytes := make([]byte, length)
		if _, err := io.ReadFull(os.Stdin, msgBytes); err != nil {
			log.Printf("failed to read message: %s", err)
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("failed to unmarshal message: %s", err)
			continue
		}

		if _, ok := msg["jsonrpc"]; !ok {
			log.Printf("invalid message format, missing jsonrpc field")
			continue
		}

		if _, ok := msg["result"]; ok {
			var response JSONRPCResponse
			if err := json.Unmarshal(msgBytes, &response); err != nil {
				log.Printf("failed to unmarshal response: %s", err)
				return
			}

			h.mu.Lock()
			responseChan, ok := h.clientChannels[response.ID]
			if !ok {
				log.Printf("no client found for ID: %s", response.ID)
				h.mu.Unlock()
				continue
			}

			responseChan <- response
			delete(h.clientChannels, response.ID)
			h.mu.Unlock()
			continue
		}

		var request JSONRPCRequest
		if err := json.Unmarshal(msgBytes, &request); err != nil {
			log.Printf("failed to unmarshal request: %s", err)
			return
		}

		if request.ID == "" {
			handler, ok := h.notificationsHandler[request.Method]
			if !ok {
				log.Printf("no handler found for notification method: %s", request.Method)
				continue
			}

			if err := handler(request.Params); err != nil {
				log.Printf("failed to handle notification %s: %s", request.Method, err)
			}
			continue
		}

		handler, ok := h.requestsHandler[request.Method]
		if !ok {
			log.Printf("no handler found for request method: %s", request.Method)
			continue
		}

		res, err := handler(request.Params)
		if err != nil {
			log.Printf("failed to handle request %s: %s", request.Method, err)
		}

		resBytes, err := json.Marshal(res)
		if err != nil {
			log.Printf("failed to marshal result: %s", err)
			continue
		}

		response := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  resBytes,
			Error:   nil,
		}

		if err := writeMessage(response); err != nil {
			log.Printf("failed to write response: %s", err)
			continue
		}
	}
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

func (h *Host) Send(request JSONRPCRequest) (JSONRPCResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.clientChannels) == 0 {
		return JSONRPCResponse{}, fmt.Errorf("no clients registered")
	}

	responseChan, ok := h.clientChannels[request.ID]
	if !ok {
		return JSONRPCResponse{}, fmt.Errorf("no client found for ID: %s", request.ID)
	}

	if err := writeMessage(request); err != nil {
		return JSONRPCResponse{}, fmt.Errorf("failed to write request: %w", err)
	}

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(10 * time.Second):
		return JSONRPCResponse{}, fmt.Errorf("timeout waiting for response")
	}
}

func (h *Host) SendNotification(method string, params any) error {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	var notification = JSONRPCRequest{
		JSONRPCVersion: "2.0",
		Method:         method,
		Params:         paramsBytes,
	}

	if err := writeMessage(notification); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}

func (h *Host) SendRequest(method string, params any, result any) error {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	id := rand.Text()

	var request = JSONRPCRequest{
		JSONRPCVersion: "2.0",
		ID:             id,
		Method:         method,
		Params:         paramsBytes,
	}

	resp, err := h.Send(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("received error response: %v", resp.Error)
	}

	if err := json.Unmarshal(resp.Result, result); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

func writeMessage(data interface{}) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	length := uint32(len(msg))

	// Write the 4-byte length header
	if err := binary.Write(os.Stdout, binary.LittleEndian, length); err != nil {
		return err
	}

	// Write the message
	_, err = os.Stdout.Write(msg)
	return err
}
