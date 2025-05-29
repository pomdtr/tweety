package main

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/go-chi/chi/v5"
	"github.com/google/shlex"
	"github.com/gorilla/websocket"

	"github.com/spf13/cobra"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

//go:embed all:themes
var themeFS embed.FS

var (
	maxBufferSizeBytes   = 512
	keepalivePingTimeout = 20 * time.Second
	activeSessions       = make(map[string]time.Time)
	sessionMutex         sync.RWMutex
)

func main() {
	var flags struct {
		host       string
		port       int
		cert       string
		key        string
		theme      string
		themeDark  string
		fontSize   int
		passphrase string
	}

	cmd := cobra.Command{
		Use:          "tweety [entrypoint]",
		Short:        "An integrated terminal for your web browser",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			themeLight := flags.theme
			themeDark := flags.themeDark
			if themeDark == "" {
				themeDark = flags.theme
			}

			handler, err := NewHandler(HandlerParams{
				Entrypoint: args[0],
				ThemeLight: themeLight,
				ThemeDark:  themeDark,
				FontSize:   flags.fontSize,
				Passphrase: flags.passphrase,
			})
			if err != nil {
				return err
			}

			if flags.cert != "" && flags.key != "" {
				cmd.PrintErrln("Listening on", fmt.Sprintf("https://%s:%d", flags.host, flags.port))
				cmd.Println("Press Ctrl+C to exit")
				return http.ListenAndServeTLS(fmt.Sprintf("%s:%d", flags.host, flags.port), flags.cert, flags.key, handler)
			}

			cmd.PrintErrln("Listening on", fmt.Sprintf("http://%s:%d", flags.host, flags.port))
			cmd.Println("Press Ctrl+C to exit")
			return http.ListenAndServe(fmt.Sprintf("%s:%d", flags.host, flags.port), handler)
		},
	}

	cmd.Flags().StringVarP(&flags.host, "host", "H", "localhost", "host to listen on")
	cmd.Flags().IntVarP(&flags.port, "port", "p", 9999, "port to listen on")
	cmd.Flags().StringVarP(&flags.cert, "cert", "c", "", "tls certificate file")
	cmd.Flags().StringVarP(&flags.key, "key", "k", "", "tls key file")
	cmd.Flags().StringVar(&flags.theme, "theme", "Tomorrow Night", "default theme to use")
	cmd.Flags().StringVar(&flags.themeDark, "theme-dark", "", "default dark theme to use, if not set, it will use the same as theme")
	cmd.Flags().IntVarP(&flags.fontSize, "font-size", "f", 13, "font size to use")
	cmd.Flags().StringVar(&flags.passphrase, "passphrase", "", "optional passphrase")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type HandlerParams struct {
	Entrypoint string
	ThemeLight string
	ThemeDark  string
	FontSize   int
	Passphrase string
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func passphraseMiddleware(passphrase string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for ping endpoint and static assets
			if r.URL.Path == "/_tweety/ping" ||
				strings.HasPrefix(r.URL.Path, "/assets/") ||
				r.URL.Path == "/icon.png" {
				next.ServeHTTP(w, r)
				return
			}

			// If no passphrase is set, let folks in
			if passphrase == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for valid session token
			if cookie, err := r.Cookie("session"); err == nil {
				sessionMutex.RLock()
				expiry, exists := activeSessions[cookie.Value]
				sessionMutex.RUnlock()

				if exists && time.Now().Before(expiry) {
					next.ServeHTTP(w, r)
					return
				}
				// If session expired or invalid, remove it
				sessionMutex.Lock()
				delete(activeSessions, cookie.Value)
				sessionMutex.Unlock()
			}

			// Handle POST request for passphrase
			if r.Method == "POST" && r.URL.Path == "/_tweety/auth" {
				if err := r.ParseForm(); err != nil {
					http.Error(w, "Bad Request", http.StatusBadRequest)
					return
				}

				providedPassphrase := r.FormValue("passphrase")
				if subtle.ConstantTimeCompare([]byte(providedPassphrase), []byte(passphrase)) == 1 {
					// Generate secure session token
					token, err := generateSecureToken()
					if err != nil {
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
						return
					}

					// Store session with expiration
					sessionMutex.Lock()
					activeSessions[token] = time.Now().Add(24 * time.Hour)
					sessionMutex.Unlock()

					// Set secure session cookie
					http.SetCookie(w, &http.Cookie{
						Name:     "session",
						Value:    token,
						Path:     "/",
						HttpOnly: true,
						Secure:   r.TLS != nil,
						SameSite: http.SameSiteStrictMode,
						MaxAge:   86400, // 24 hours
					})
					w.WriteHeader(http.StatusOK)
					return
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// If no valid session, show the form
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`
					<!DOCTYPE html>
					<html lang="en">
					<head>
						<title>Tweety Access</title>
						<style>
							body { font-family: system-ui; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
							.container { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
							input { padding: 0.5rem; margin: 0.5rem 0; width: 100%; }
							button { padding: 0.5rem 1rem; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; }
							.error { color: #dc3545; margin-top: 1rem; display: none; }
						</style>
					</head>
					<body>
						<div class="container">
							<h1>Tweety Access</h1>
							<form id="passphraseForm">
								<input type="password" id="passphrase" placeholder="Enter passphrase" required>
								<button type="submit">Access Tweety</button>
							</form>
							<div id="error" class="error">Incorrect passphrase. Try again.</div>
						</div>
						<script>
							document.getElementById('passphraseForm').onsubmit = async function(e) {
								e.preventDefault();
								const passphrase = document.getElementById('passphrase').value;
								
								try {
									const response = await fetch('/_tweety/auth', {
										method: 'POST',
										headers: {
											'Content-Type': 'application/x-www-form-urlencoded',
										},
										body: 'passphrase=' + encodeURIComponent(passphrase)
									});

									if (response.ok) {
										window.location.href = '/';
									} else {
										document.getElementById('error').style.display = 'block';
									}
								} catch (error) {
									console.error('Error:', error);
									document.getElementById('error').style.display = 'block';
								}
							};
						</script>
					</body>
					</html>
				`))
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}

func NewHandler(params HandlerParams) (http.Handler, error) {
	r := chi.NewRouter()

	// Passphrase middleware
	r.Use(passphraseMiddleware(params.Passphrase))

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

	r.Get("/_tweety/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	ttyMap := make(map[string]*os.File)

	r.Post("/_tweety/exec", func(w http.ResponseWriter, r *http.Request) {
		var args []string
		if cmd := r.URL.Query().Get("cmd"); cmd != "" {
			a, err := shlex.Split(cmd)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(fmt.Sprintf("invalid command: %s", err)))
				return
			}

			args = a
		}

		cmd := exec.Command(params.Entrypoint, args...)

		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "TERM=xterm-256color")

		if cwd := r.URL.Query().Get("cwd"); cwd != "" {
			cmd.Dir = cwd
		}

		log.Println("executing command:", cmd.String())
		tty, err := pty.Start(cmd)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		id := rand.Text()
		ttyMap[id] = tty

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(id))
	})

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

	r.Post("/_tweety/resize/{terminalID}", func(w http.ResponseWriter, r *http.Request) {
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

	frontendFS, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		return nil, err
	}

	index, err := template.ParseFS(frontendFS, "index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse index.html: %w", err)
	}

	themeBytes, err := themeFS.ReadFile(fmt.Sprintf("themes/%s.json", strings.Trim(params.ThemeLight, " ")))
	if err != nil {
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}

	themeDarkBytes, err := themeFS.ReadFile(fmt.Sprintf("themes/%s.json", strings.Trim(params.ThemeDark, " ")))
	if err != nil {
		return nil, fmt.Errorf("failed to read dark theme file: %w", err)
	}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		index.Execute(w, map[string]interface{}{
			"ThemeLight": string(themeBytes),
			"ThemeDark":  string(themeDarkBytes),
			"FontSize":   params.FontSize,
		})
	})

	r.Handle("/*", http.FileServer(http.FS(frontendFS)))

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
) *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		HandshakeTimeout: 0,
		ReadBufferSize:   maxBufferSizeBytes,
		WriteBufferSize:  maxBufferSizeBytes,
	}
}
