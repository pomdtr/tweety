package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
)

func NewHandler() (http.Handler, error) {
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

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://local.tweety.sh", "chrome-extension://*"},
	}))

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
		config, err := LoadConfig(configPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		config.Env = nil
		for k, profile := range config.Profiles {
			profile.Command = ""
			profile.Args = nil
			profile.Cwd = ""
			profile.Env = nil
			config.Profiles[k] = profile
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)

		if err := encoder.Encode(config); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	})

	ttyMap := make(map[string]*os.File)
	r.Get("/pty/{terminalID}", func(w http.ResponseWriter, r *http.Request) {
		terminalID := chi.URLParam(r, "terminalID")
		config, err := LoadConfig(configPath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

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

		profileName := r.URL.Query().Get("profile")
		profile, ok := config.Profiles[profileName]
		if !ok {
			log.Println("invalid profile name:", profileName)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid profile name: %s", profileName)))
			return
		}

		cmd := exec.Command(profile.Command, profile.Args...)
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")
		for k, v := range config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		for k, v := range profile.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("failed to get user home directory: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if profile.Cwd != "" {
			if profile.Cwd == "~" {
				cmd.Dir = homeDir
			} else if strings.HasPrefix(profile.Cwd, "~/") {
				cmd.Dir = filepath.Join(homeDir, profile.Cwd[2:])
			} else {
				cmd.Dir = profile.Cwd
			}
		} else {
			cmd.Dir = homeDir
		}

		log.Println("executing command:", cmd.String())
		tty, err := pty.Start(cmd)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		defer func() {
			log.Print("killing spawned process")
			cmd.Process.Kill()
			if err := tty.Close(); err != nil {
				log.Printf("failed to close spawned tty gracefully: %s", err)
			}
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

	r.Post("/resize/{terminalID}", func(w http.ResponseWriter, r *http.Request) {
		terminalID := chi.URLParam(r, "terminalID")
		tty, ok := ttyMap[terminalID]

		if !ok {
			availableTerminals := make([]string, 0, len(ttyMap))
			for k := range ttyMap {
				availableTerminals = append(availableTerminals, k)
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid terminal ID: %s, available terminal: %v", terminalID, availableTerminals)))
			return
		}

		var resizePayload struct {
			Rows uint16 `json:"rows"`
			Cols uint16 `json:"cols"`
		}
		if err := json.NewDecoder(r.Body).Decode(&resizePayload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid resize payload: %s", err)))
			return
		}

		if err := pty.Setsize(tty, &pty.Winsize{
			Rows: resizePayload.Rows,
			Cols: resizePayload.Cols,
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.Write([]byte("Resized"))
	})

	themeHandler, err := ThemeHandler()
	if err != nil {
		return nil, err
	}

	r.Handle("/themes/*", http.StripPrefix("/themes", themeHandler))

	frontendHandler, err := FrontendHandler()
	if err != nil {
		return nil, err
	}
	r.Handle("/*", frontendHandler)

	return r, nil
}

//go:embed all:frontend/dist
var frontendDist embed.FS

func FrontendHandler() (http.Handler, error) {
	fs, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		return nil, err
	}

	return http.FileServer(http.FS(fs)), nil
}

//go:embed all:themes
var themes embed.FS

func ThemeHandler() (http.Handler, error) {
	fs, err := fs.Sub(themes, "themes")
	if err != nil {
		return nil, err
	}

	return http.FileServer(http.FS(fs)), nil
}

var (
	maxBufferSizeBytes   = 512
	keepalivePingTimeout = 20 * time.Second
)

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

var WebsocketMessageType = map[int]string{
	websocket.BinaryMessage: "binary",
	websocket.TextMessage:   "text",
	websocket.CloseMessage:  "close",
	websocket.PingMessage:   "ping",
	websocket.PongMessage:   "pong",
}
