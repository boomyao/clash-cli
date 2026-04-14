package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the HTTP client for mihomo's RESTful API.
type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(externalController, secret string) *Client {
	return &Client{
		baseURL: "http://" + externalController,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// BaseURL returns the API base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Secret returns the API secret.
func (c *Client) Secret() string {
	return c.secret
}

// WSBaseURL returns the WebSocket base URL.
func (c *Client) WSBaseURL() string {
	return "ws://" + c.baseURL[len("http://"):]
}

// do executes an HTTP request with auth header.
func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// get performs a GET request and decodes the JSON response.
func (c *Client) get(path string, result interface{}) error {
	resp, err := c.do("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// put performs a PUT request.
func (c *Client) put(path string, body interface{}) error {
	resp, err := c.do("PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// patch performs a PATCH request.
func (c *Client) patch(path string, body interface{}) error {
	resp, err := c.do("PATCH", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// post performs a POST request.
func (c *Client) post(path string, body interface{}) error {
	resp, err := c.do("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// delete performs a DELETE request.
func (c *Client) delete(path string) error {
	resp, err := c.do("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
