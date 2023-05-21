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
	"unsafe"

	"github.com/google/uuid"
)

// nativeEndian used to detect native byte order
var nativeEndian binary.ByteOrder

var webTermPort = 9999

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

func NewServer(m *MessageHandler, environ []string) *http.Server {
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
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

		msg, err := m.send(payload)
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

	var command string
	var args []string
	if shell, ok := os.LookupEnv("SHELL"); ok {
		command = shell
		args = []string{"-li"}
	} else {
		command = "/bin/bash"
		args = []string{"bash", "-li"}
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/pty", WebSocketHandler(HandlerOpts{
		Command: command,
		Args:    args,
		Env:     environ,
		Dir:     dir,
	}))

	log.Printf("Listening on port %d\n", webTermPort)

	return &http.Server{
		Addr: fmt.Sprintf(":%d", webTermPort),
	}
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

func (h *MessageHandler) send(payload any) (any, error) {
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
