package jsonrpc

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
)

type JSONRPCClient struct {
	Port  int
	Token string
}

func NewClient(port int, token string) *JSONRPCClient {
	return &JSONRPCClient{
		Port:  port,
		Token: token,
	}
}

func (c *JSONRPCClient) SendRequest(method string, params interface{}) (*JSONRPCResponse, error) {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	body, err := json.Marshal(JSONRPCRequest{
		JSONRPCVersion: "2.0",
		ID:             rand.Text(),
		Method:         method,
		Params:         paramsBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", c.Port), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}
