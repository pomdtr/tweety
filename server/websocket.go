package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

const DefaultConnectionErrorLimit = 10

type HandlerOpts struct {
	Command string
	Args    []string
	Env     []string
	Dir     string

	// ConnectionErrorLimit defines the number of consecutive errors that can happen
	// before a connection is considered unusable
	ConnectionErrorLimit int
	// KeepalivePingTimeout defines the maximum duration between which a ping and pong
	// cycle should be tolerated, beyond this the connection should be deemed dead
	KeepalivePingTimeout time.Duration
	MaxBufferSizeBytes   int
}

func WebSocketHandler(opts HandlerOpts) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

		tabUrl := r.URL.Query().Get("url")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid tabUrl value: %s", err)))
			return
		}

		environ := opts.Env
		environ = append(environ, fmt.Sprintf("TAB_URL=%s", tabUrl))

		log.Printf("received connection request with cols=%d, rows=%d", cols, rows)

		connectionErrorLimit := opts.ConnectionErrorLimit
		if connectionErrorLimit < 0 {
			connectionErrorLimit = DefaultConnectionErrorLimit
		}
		maxBufferSizeBytes := opts.MaxBufferSizeBytes
		if maxBufferSizeBytes == 0 {
			maxBufferSizeBytes = 512
		}
		keepalivePingTimeout := opts.KeepalivePingTimeout
		if keepalivePingTimeout <= time.Second {
			keepalivePingTimeout = 20 * time.Second
		}

		log.Print("established connection identity")
		upgrader := getConnectionUpgrader(maxBufferSizeBytes)
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("failed to upgrade connection: %s", err)
			return
		}

		cmd := exec.Command(opts.Command, opts.Args...)
		cmd.Env = environ
		cmd.Dir = opts.Dir
		tty, err := pty.Start(cmd)
		if err := pty.Setsize(tty, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		defer func() {
			log.Print("gracefully stopping spawned tty...")

			if err := tty.Close(); err != nil {
				log.Printf("failed to close spawned tty gracefully: %s", err)
			}
			if err := connection.Close(); err != nil {
				log.Printf("failed to close webscoket connection: %s", err)
			}
		}()

		var connectionClosed bool
		var waiter sync.WaitGroup
		waiter.Add(1)

		// this is a keep-alive loop that ensures connection does not hang-up itself
		lastPongTime := time.Now()
		connection.SetPongHandler(func(msg string) error {
			lastPongTime = time.Now()
			return nil
		})
		go func() {
			for {
				if err := connection.WriteMessage(websocket.PingMessage, []byte("keepalive")); err != nil {
					log.Printf("failed to write ping message")
					return
				}
				time.Sleep(keepalivePingTimeout / 2)
				if time.Since(lastPongTime) > keepalivePingTimeout {
					log.Printf("failed to get response from ping, triggering disconnect now...")
					waiter.Done()
					return
				}
				log.Printf("received response from ping successfully")
			}
		}()

		// tty >> xterm.js
		go func() {
			errorCounter := 0
			for {
				// consider the connection closed/errored out so that the socket handler
				// can be terminated - this frees up memory so the service doesn't get
				// overloaded
				if errorCounter > connectionErrorLimit {
					waiter.Done()
					break
				}
				buffer := make([]byte, maxBufferSizeBytes)
				readLength, err := tty.Read(buffer)
				if err != nil {
					log.Printf("failed to read from tty: %s", err)
					waiter.Done()
					return
				}
				if err := connection.WriteMessage(websocket.BinaryMessage, buffer[:readLength]); err != nil {
					log.Printf("failed to send %v bytes from tty to xterm.js", readLength)
					errorCounter++
					continue
				}
				errorCounter = 0
			}
		}()

		// tty << xterm.js
		go func() {
			for {
				// data processing
				_, data, err := connection.ReadMessage()
				if err != nil {
					if !connectionClosed {
						log.Printf("failed to get next reader: %s", err)
					}
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

		waiter.Wait()
		log.Printf("closing connection...")
		connectionClosed = true
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
