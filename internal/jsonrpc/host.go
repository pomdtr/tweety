package jsonrpc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

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

func (h *Host) Send(request JSONRPCRequest) (JSONRPCResponse, error) {
	h.mu.Lock()
	responseChan := make(chan JSONRPCResponse)
	h.clientChannels[request.ID] = responseChan
	h.mu.Unlock()

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
