package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
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

func main() {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "tweety", "config.json")

	cmd := &cobra.Command{
		Use:   "tweety",
		Short: "An integrated terminal for your web browser",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := k.Load(file.Provider(configPath), jsonparser.Parser()); err != nil {
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
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := getFreePort()
			if err != nil {
				return fmt.Errorf("failed to get free port: %w", err)
			}

			token := rand.Text()

			handler, err := NewHandler(HandlerParams{
				Entrypoint: args[0],
				Port:       port,
				Token:      token,
			})
			if err != nil {
				return err
			}

			cmd.PrintErrf("Listening on http://localhost:%d\n", port)
			cmd.Println("Press Ctrl+C to exit")
			return http.ListenAndServe(fmt.Sprintf(":%d", port), handler)
		},
	}

	return cmd
}

func NewCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use: "install",
	}

	return cmd
}

type HandlerParams struct {
	Entrypoint string
	Port       int
	Token      string
}

func NewHandler(params HandlerParams) (http.Handler, error) {
	r := chi.NewRouter()

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

	// r.Get("/_tweety/ping", func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("pong"))
	// })

	ttyMap := make(map[string]*os.File)

	// r.Post("/_tweety/exec", func(w http.ResponseWriter, r *http.Request) {
	// 	var args []string
	// 	if param := r.URL.Query().Get("args"); param != "" {
	// 		a, err := shlex.Split(param)
	// 		if err != nil {
	// 			w.WriteHeader(http.StatusBadRequest)
	// 			w.Write([]byte(fmt.Sprintf("invalid command: %s", err)))
	// 			return
	// 		}

	// 		args = a
	// 	}

	// 	cmd := exec.Command(params.Entrypoint, args...)

	// 	cmd.Env = os.Environ()
	// 	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	// 	cmd.Env = append(cmd.Env, fmt.Sprintf("TWEETY_PORT=%d", params.Port))
	// 	cmd.Env = append(cmd.Env, fmt.Sprintf("TWEETY_TOKEN=%s", params.Token))

	// 	if cwd := r.URL.Query().Get("cwd"); cwd != "" {
	// 		cmd.Dir = cwd
	// 	}

	// 	log.Println("executing command:", cmd.String())
	// 	tty, err := pty.Start(cmd)
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		w.Write([]byte(err.Error()))
	// 		return
	// 	}

	// 	id := rand.Text()
	// 	ttyMap[id] = tty

	// 	w.Header().Set("Content-Type", "text/plain")
	// 	w.WriteHeader(http.StatusOK)
	// 	w.Write([]byte(id))
	// })

	r.Get("/_tweety/pty/{terminalID}", func(w http.ResponseWriter, r *http.Request) {
		terminalID := chi.URLParam(r, "terminalID")
		cols, err := strconv.Atoi(r.URL.Query().Get("cols"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid cols value: %s", err)))
			return
		}

		rows, err := strconv.Atoi(r.URL.Query().Get("rows"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid rows value: %s", err)))
			return
		}

		tty := ttyMap[terminalID]
		if tty == nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid terminal ID: %s", terminalID)))
			return
		}

		defer func() {
			// cleanup
			delete(ttyMap, terminalID)
			// send the signal to the process
			tty.Close()
		}()

		if err := pty.Setsize(tty, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		ttyMap[terminalID] = tty
		HandleWebsocket(tty)(w, r)
	})

	// r.Post("/_tweety/resize/{terminalID}", func(w http.ResponseWriter, r *http.Request) {
	// 	terminalID := chi.URLParam(r, "terminalID")
	// 	tty, ok := ttyMap[terminalID]

	// 	if !ok {
	// 		availableTerminals := make([]string, 0, len(ttyMap))
	// 		for k := range ttyMap {
	// 			availableTerminals = append(availableTerminals, k)
	// 		}

	// 		w.WriteHeader(http.StatusBadRequest)
	// 		w.Write([]byte(fmt.Sprintf("invalid terminal ID: %s, available terminal: %v", terminalID, availableTerminals)))
	// 		return
	// 	}

	// 	var resizePayload struct {
	// 		Rows uint16 `json:"rows"`
	// 		Cols uint16 `json:"cols"`
	// 	}
	// 	if err := json.NewDecoder(r.Body).Decode(&resizePayload); err != nil {
	// 		w.WriteHeader(http.StatusBadRequest)
	// 		w.Write([]byte(fmt.Sprintf("invalid resize payload: %s", err)))
	// 		return
	// 	}

	// 	if err := pty.Setsize(tty, &pty.Winsize{
	// 		Rows: resizePayload.Rows,
	// 		Cols: resizePayload.Cols,
	// 	}); err != nil {
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		w.Write([]byte(err.Error()))
	// 		return
	// 	}

	// 	w.Write([]byte("Resized"))
	// })

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
