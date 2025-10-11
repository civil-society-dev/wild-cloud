// Package client provides HTTP client for Wild Central daemon API
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the HTTP client for the Wild Central daemon
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIResponse is the API response format
type APIResponse struct {
	Data  map[string]interface{}
	Error string `json:"error,omitempty"`
}

// Get makes a GET request to the API
func (c *Client) Get(path string) (*APIResponse, error) {
	return c.doRequest("GET", path, nil)
}

// Post makes a POST request to the API
func (c *Client) Post(path string, body interface{}) (*APIResponse, error) {
	return c.doRequest("POST", path, body)
}

// Put makes a PUT request to the API
func (c *Client) Put(path string, body interface{}) (*APIResponse, error) {
	return c.doRequest("PUT", path, body)
}

// Delete makes a DELETE request to the API
func (c *Client) Delete(path string) (*APIResponse, error) {
	return c.doRequest("DELETE", path, nil)
}

// Patch makes a PATCH request to the API
func (c *Client) Patch(path string, body interface{}) (*APIResponse, error) {
	return c.doRequest("PATCH", path, body)
}

// doRequest performs the actual HTTP request
func (c *Client) doRequest(method, path string, body interface{}) (*APIResponse, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP error status
	if resp.StatusCode >= 400 {
		// Try to parse error response
		var errResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errResp); err == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				return nil, fmt.Errorf("API error: %s", errMsg)
			}
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response data directly (daemon doesn't wrap in "data" field)
	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w\nResponse: %s", err, string(respBody))
	}

	return &APIResponse{Data: data}, nil
}

// GetData extracts data from API response
func (r *APIResponse) GetData(key string) interface{} {
	if r.Data == nil {
		return nil
	}
	return r.Data[key]
}

// GetString extracts string data from API response
func (r *APIResponse) GetString(key string) string {
	val := r.GetData(key)
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// GetMap extracts map data from API response
func (r *APIResponse) GetMap(key string) map[string]interface{} {
	val := r.GetData(key)
	if m, ok := val.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// GetArray extracts array data from API response
func (r *APIResponse) GetArray(key string) []interface{} {
	val := r.GetData(key)
	if arr, ok := val.([]interface{}); ok {
		return arr
	}
	return nil
}

// BaseURL returns the base URL of the client
func (c *Client) BaseURL() string {
	return c.baseURL
}
