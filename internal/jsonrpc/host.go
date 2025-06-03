package jsonrpc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type RequestHandlerFunc = func(params []byte) (result any, err error)
type NotificationHandlerFunc = func(params []byte) error

type Host struct {
	logger               *slog.Logger
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

func NewHost(logger *slog.Logger) *Host {
	return &Host{
		logger:               logger,
		requestsHandler:      make(map[string]RequestHandlerFunc),
		notificationsHandler: make(map[string]NotificationHandlerFunc),
		clientChannels:       make(map[string]chan JSONRPCResponse),
	}
}

func (h *Host) Listen() error {
	for {
		lengthBytes := make([]byte, 4)
		if _, err := io.ReadFull(os.Stdin, lengthBytes); err != nil {
			if err == io.EOF {
				h.logger.Info("EOF reached, stopping listener")
				return nil
			}

			h.logger.Error("failed to read message length", "error", err)
			return err
		}

		length := binary.LittleEndian.Uint32(lengthBytes)

		msgBytes := make([]byte, length)
		if _, err := io.ReadFull(os.Stdin, msgBytes); err != nil {
			h.logger.Error("failed to read message", "error", err)
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			h.logger.Error("failed to unmarshal message", "error", err)
			continue
		}

		if _, ok := msg["jsonrpc"]; !ok {
			h.logger.Error("invalid message format, missing jsonrpc field")
			continue
		}

		_, containsResult := msg["result"]
		_, containsError := msg["error"]

		if containsResult || containsError {
			var response JSONRPCResponse
			if err := json.Unmarshal(msgBytes, &response); err != nil {
				h.logger.Error("failed to unmarshal response", "error", err)
				return err
			}

			h.mu.Lock()
			responseChan, ok := h.clientChannels[response.ID]
			if !ok {
				h.logger.Error("no client found")
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
			h.logger.Error("failed to unmarshal request", "error", err)
			return err
		}

		if request.ID == "" {
			handler, ok := h.notificationsHandler[request.Method]
			if !ok {
				h.logger.Error("no handler found for notification", "method", request.Method)
				continue
			}

			go func() {
				if err := handler(request.Params); err != nil {
					h.logger.Error("failed to handle notification", "method", request.Method, "error", err)
				}
			}()

			continue
		}

		handler, ok := h.requestsHandler[request.Method]
		if !ok {
			h.logger.Error("no handler found for request", "method", request.Method)

			errorBytes, err := json.Marshal(map[string]any{
				"code":    -32601,
				"message": fmt.Sprintf("Method not found: %s", request.Method),
			})
			if err != nil {
				h.logger.Error("failed to marshal error response", "error", err)
				continue
			}

			writeMessage(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error:   errorBytes,
			})
			continue
		}

		go func() {
			res, err := handler(request.Params)
			if err != nil {
				h.logger.Error("failed to handle request", "method", request.Method, "err", err)
				errorBytes, err := json.Marshal(map[string]any{
					"code":    -32603,
					"message": fmt.Sprintf("Internal error: %s", err),
				})
				if err != nil {
					h.logger.Error("failed to marshal error response", "error", err)
					return
				}

				writeMessage(JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Error:   errorBytes,
				})
				return
			}

			resBytes, err := json.Marshal(res)
			if err != nil {
				h.logger.Error("failed to marshal result", "error", err)
				return
			}

			response := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result:  resBytes,
				Error:   nil,
			}

			if err := writeMessage(response); err != nil {
				h.logger.Error("failed to write response", "error", err)
				return
			}
		}()
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
	case <-time.After(5 * time.Second):
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
