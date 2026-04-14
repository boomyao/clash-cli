package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// GetProxies fetches all proxies.
func (c *Client) GetProxies() (*ProxiesResponse, error) {
	var result ProxiesResponse
	if err := c.get("/proxies", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProxy fetches details for a single proxy.
func (c *Client) GetProxy(name string) (*Proxy, error) {
	var result Proxy
	path := fmt.Sprintf("/proxies/%s", url.PathEscape(name))
	if err := c.get(path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SelectProxy selects a proxy node within a group.
func (c *Client) SelectProxy(group, proxy string) error {
	path := fmt.Sprintf("/proxies/%s", url.PathEscape(group))
	return c.put(path, SelectProxyRequest{Name: proxy})
}

// TestProxyDelay tests the latency of a single proxy.
func (c *Client) TestProxyDelay(name string, testURL string, timeout int) (int, error) {
	if testURL == "" {
		testURL = "https://www.gstatic.com/generate_204"
	}
	if timeout <= 0 {
		timeout = 5000
	}

	path := fmt.Sprintf("/proxies/%s/delay?url=%s&timeout=%d",
		url.PathEscape(name),
		url.QueryEscape(testURL),
		timeout,
	)

	resp, err := c.do("GET", path, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Delay   int    `json:"delay"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("decode delay response: %w", err)
	}

	if result.Message != "" {
		return 0, fmt.Errorf("delay test failed: %s", result.Message)
	}

	return result.Delay, nil
}
