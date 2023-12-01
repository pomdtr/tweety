package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pomdtr/popcorn/internal/config"
)

// nativeEndian used to detect native byte order
var nativeEndian binary.ByteOrder

func init() {
	// determine native byte order so that we can read message size correctly
	var one int16 = 1
	b := (*byte)(unsafe.Pointer(&one))
	if *b == 0 {
		nativeEndian = binary.BigEndian
	} else {
		nativeEndian = binary.LittleEndian
	}
}

type ExtensionMessage struct {
	ID      string `json:"id"`
	Payload any    `json:"payload"`
	Error   string `json:"error,omitempty"`
}

// readMessageLength reads and returns the message length value in native byte order.
func readMessageLength(msg []byte) (int, error) {
	var length uint32
	buf := bytes.NewBuffer(msg)
	err := binary.Read(buf, nativeEndian, &length)
	if err != nil {
		return 0, fmt.Errorf("unable to read bytes representing message length: %w", err)
	}

	return int(length), nil
}

func Serve(m *MessageHandler, port int, token string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/browser", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
			return
		}

		var payload any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		msg, err := m.SendMessage(payload)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(msg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	})

	ttyMap := make(map[string]*os.File)
	http.HandleFunc("/pty/", WebSocketHandler(HandlerOpts{
		Token: token,
		Env: []string{
			fmt.Sprintf("POPCORN_PORT=%d", port),
		},
		ttyMap: ttyMap,
	}))

	http.HandleFunc("/resize/", func(w http.ResponseWriter, r *http.Request) {
		terminalID := strings.TrimPrefix(r.URL.Path, "/resize/")
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

		if err := pty.Setsize(tty, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Resized"))
	})

	log.Println("Listening on port", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

}

type Message struct {
	content any
	err     error
}

type MessageHandler struct {
	subscriptions map[string]chan Message
}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		subscriptions: make(map[string]chan Message),
	}
}

func (h *MessageHandler) SendMessage(payload any) (any, error) {
	msgID := uuid.New().String()

	msg := ExtensionMessage{
		ID:      msgID,
		Payload: payload,
	}

	byteMsg, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal OutgoingMessage struct to slice of bytes: %w", err)
	}

	log.Printf("Sending message: %s", string(byteMsg))
	if err := binary.Write(os.Stdout, nativeEndian, uint32(len(byteMsg))); err != nil {
		return nil, fmt.Errorf("unable to write message length to Stdout: %v", err)
	}

	var msgBuf bytes.Buffer
	if _, err := msgBuf.Write(byteMsg); err != nil {
		return nil, fmt.Errorf("unable to write message to buffer: %w", err)
	}

	c := make(chan Message)
	h.subscriptions[msgID] = c
	_, err = msgBuf.WriteTo(os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("unable to write message buffer to Stdout: %w", err)
	}

	out := <-c
	delete(h.subscriptions, msgID)
	if out.err != nil {
		return nil, out.err
	}

	return out.content, nil
}

func (h *MessageHandler) Loop() {
	for {
		v := bufio.NewReader(os.Stdin)
		// adjust buffer size to accommodate your json payload size limits; default is 4096
		s := bufio.NewReader(v)

		lengthBytes := make([]byte, 4)

		// we're going to indefinitely read the first 4 bytes in buffer, which gives us the message length.
		// if stdIn is closed we'll exit the loop and shut down host
		if _, err := s.Read(lengthBytes); err != nil {
			if err == io.EOF {
				log.Printf("Stdin closed; shutting down host")
				os.Exit(0)
			}

			log.Printf("Error reading from Stdin: %v", err)
			continue
		}

		// convert message length bytes to integer value
		lengthNum, err := readMessageLength(lengthBytes)
		log.Printf("Message length: %d", lengthNum)
		if err != nil {
			log.Printf("Error converting message length bytes to integer value: %v", err)
			continue
		}

		// read the content of the message from buffer
		content := make([]byte, lengthNum)
		log.Printf("Reading message content from buffer")
		if _, err := io.ReadFull(s, content); err != nil {
			log.Printf("Error reading message content from buffer: %v", err)
			continue
		}

		var msg ExtensionMessage
		// unmarshal message to JSON
		if err := json.Unmarshal(content, &msg); err != nil {
			log.Printf("Error unmarshalling message to JSON: %v", err)
			continue
		}

		c, ok := h.subscriptions[msg.ID]
		if !ok {
			log.Printf("No subscription found for message ID: %s", msg.ID)
			continue
		}

		if msg.Error != "" {
			c <- Message{
				err: fmt.Errorf(msg.Error),
			}
			continue
		}

		c <- Message{
			content: msg.Payload,
		}
	}
}

const DefaultConnectionErrorLimit = 10

type HandlerOpts struct {
	Env    []string
	Token  string
	ttyMap map[string]*os.File

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
		terminalID := strings.TrimPrefix(r.URL.Path, "/pty/")
		token := r.URL.Query().Get("token")

		if token != opts.Token {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("unauthorized"))
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

		cfg, err := config.Load(config.Path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		profileName := r.URL.Query().Get("profile")
		profile, ok := cfg.Profiles[profileName]
		if !ok {
			log.Println("invalid profile name:", profileName)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid profile name: %s", profileName)))
			return
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("failed to get user home directory: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		cmd := exec.Command(profile.Command, profile.Args...)
		cmd.Env = append(cmd.Env, opts.Env...)
		cmd.Env = append(cmd.Env, "TERM=xterm-256color", fmt.Sprintf("POPCORN_TERMINAL_ID=%s", terminalID))
		for k, v := range profile.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		cmd.Dir = homeDir

		log.Println("executing command:", cmd.String())

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

		tty, err := pty.Start(cmd)
		opts.ttyMap[terminalID] = tty
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
		delete(opts.ttyMap, terminalID)
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
