package jsonrpc

import "encoding/json"

type JSONRPCRequest struct {
	JSONRPCVersion string          `json:"jsonrpc"`
	ID             string          `json:"id,omitempty"`
	Method         string          `json:"method"`
	Params         json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}
