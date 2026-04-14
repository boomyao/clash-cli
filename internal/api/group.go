package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// GetGroups fetches all proxy groups.
func (c *Client) GetGroups() (map[string]Proxy, error) {
	var result ProxyGroupsResponse
	if err := c.get("/group", &result); err != nil {
		return nil, err
	}
	return result.Proxies, nil
}

// GetGroup fetches details for a single proxy group.
func (c *Client) GetGroup(name string) (*Proxy, error) {
	var result Proxy
	path := fmt.Sprintf("/group/%s", url.PathEscape(name))
	if err := c.get(path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TestGroupDelay tests the latency of all proxies in a group.
func (c *Client) TestGroupDelay(name string, testURL string, timeout int) (GroupDelayResponse, error) {
	if testURL == "" {
		testURL = "https://www.gstatic.com/generate_204"
	}
	if timeout <= 0 {
		timeout = 5000
	}

	path := fmt.Sprintf("/group/%s/delay?url=%s&timeout=%d",
		url.PathEscape(name),
		url.QueryEscape(testURL),
		timeout,
	)

	resp, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result GroupDelayResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode group delay response: %w", err)
	}

	return result, nil
}
