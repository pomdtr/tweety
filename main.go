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

type Request struct {
	JSONRPCVersion string          `json:"jsonrpc"`
	ID             string          `json:"id"`
	Method         string          `json:"method"`
	Params         json.RawMessage `json:"params"`
}

type RequestParamsExec struct {
	Cwd  string   `json:"cwd"`
	Args []string `json:"args"`
	Rows uint16   `json:"rows"`
	Cols uint16   `json:"cols"`
}

type RequestParamsResize struct {
	ID   string `json:"id"`
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
				Command: k.String("command"),
				Port:    port,
				Token:   token,
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
			hostFile, err := os.Create(hostPath)
			if err != nil {
				return fmt.Errorf("failed to create native messaging host file: %w", err)
			}
			defer hostFile.Close()

			execPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get executable path: %w", err)
			}

			if err := hostTemplate.Execute(hostFile, map[string]interface{}{
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

			manifestFile, err := os.Create(filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome", "NativeMessagingHosts", "com.github.pomdtr.tweety.json"))
			if err != nil {
				return fmt.Errorf("failed to get manifest file path: %w", err)
			}
			defer manifestFile.Close()

			if err := manifestTemplate.Execute(manifestFile, map[string]interface{}{
				"Path": hostPath,
			}); err != nil {
				return fmt.Errorf("failed to execute manifest template: %w", err)
			}

			return nil
		},
	}

	return cmd
}

type HandlerParams struct {
	Command string
	Port    int
	Token   string
}

func NewHandler(handlerParams HandlerParams) (http.Handler, error) {
	r := chi.NewRouter()

	var ttyMap = make(map[string]*os.File)
	go func() {
		for {
			msg, err := readNativeMessage()
			if err != nil {
				log.Printf("failed to read message: %s", err)
				continue
			}

			var request Request
			if err := json.Unmarshal(msg, &request); err != nil {
				log.Printf("failed to unmarshal message: %s", err)
				continue
			}

			log.Printf("received request: %s", request.Method)
			switch request.Method {
			case "exec":
				var requestParams RequestParamsExec
				if err := json.Unmarshal(request.Params, &requestParams); err != nil {
					log.Printf("failed to unmarshal exec params: %s", err)
					continue
				}

				entrypoint := k.String("command")
				if _, err := exec.LookPath(entrypoint); err != nil && !filepath.IsAbs(entrypoint) {
					entrypoint = filepath.Join(configDir, entrypoint)
				}

				cmd := exec.Command(entrypoint, requestParams.Args...)

				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, "TERM=xterm-256color")
				cmd.Env = append(cmd.Env, fmt.Sprintf("TWEETY_PORT=%d", handlerParams.Port))
				cmd.Env = append(cmd.Env, fmt.Sprintf("TWEETY_TOKEN=%s", handlerParams.Token))
				for key, value := range k.StringMap("env") {
					cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
				}

				if requestParams.Cwd != "" {
					cmd.Dir = requestParams.Cwd
				} else {
					cmd.Dir = os.Getenv("HOME")
				}

				log.Println("executing command:", cmd.String())
				tty, err := pty.Start(cmd)
				if err != nil {
					log.Printf("failed to start pty: %s", err)
					return
				}

				if err := pty.Setsize(tty, &pty.Winsize{Rows: requestParams.Rows, Cols: requestParams.Cols}); err != nil {
					log.Printf("failed to set size for tty: %s", err)
					return
				}

				ttyID := strings.ToLower(rand.Text())
				ttyMap[ttyID] = tty

				writeMessage(map[string]interface{}{
					"id":      request.ID,
					"jsonrpc": "2.0",
					"result": map[string]string{
						"url": fmt.Sprintf("ws://127.0.0.1:%d/tty/%s", handlerParams.Port, ttyID),
						"id":  ttyID,
					},
				})
			case "resize":
				var requestParams RequestParamsResize
				if err := json.Unmarshal(request.Params, &requestParams); err != nil {
					log.Printf("failed to unmarshal resize params: %s", err)
					continue
				}

				tty, ok := ttyMap[requestParams.ID]
				if !ok {
					log.Printf("no tty found for ID: %s", requestParams.ID)
					continue
				}

				if err := pty.Setsize(tty, &pty.Winsize{Rows: requestParams.Rows, Cols: requestParams.Cols}); err != nil {
					log.Printf("failed to set size for tty %s: %s", requestParams.ID, err)
					continue
				}
			}
		}
	}()

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

func readNativeMessage() ([]byte, error) {
	lengthBytes := make([]byte, 4)
	if _, err := io.ReadFull(os.Stdin, lengthBytes); err != nil {
		return nil, fmt.Errorf("failed to read length header: %w", err)
	}
	length := binary.LittleEndian.Uint32(lengthBytes)

	messageBytes := make([]byte, length)
	if _, err := io.ReadFull(os.Stdin, messageBytes); err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	return messageBytes, nil
}
