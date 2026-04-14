package api

import (
	"fmt"
	"net/url"
)

// GetProxyProviders fetches all proxy providers.
func (c *Client) GetProxyProviders() (*ProvidersResponse, error) {
	var result ProvidersResponse
	if err := c.get("/providers/proxies", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateProxyProvider refreshes a proxy provider.
func (c *Client) UpdateProxyProvider(name string) error {
	path := fmt.Sprintf("/providers/proxies/%s", url.PathEscape(name))
	return c.put(path, nil)
}

// HealthCheckProvider performs a health check on a proxy provider.
func (c *Client) HealthCheckProvider(name string) error {
	path := fmt.Sprintf("/providers/proxies/%s/healthcheck", url.PathEscape(name))
	_, err := c.do("GET", path, nil)
	return err
}

// GetRuleProviders fetches all rule providers.
func (c *Client) GetRuleProviders() (map[string]Provider, error) {
	var result struct {
		Providers map[string]Provider `json:"providers"`
	}
	if err := c.get("/providers/rules", &result); err != nil {
		return nil, err
	}
	return result.Providers, nil
}

// UpdateRuleProvider refreshes a rule provider.
func (c *Client) UpdateRuleProvider(name string) error {
	path := fmt.Sprintf("/providers/rules/%s", url.PathEscape(name))
	return c.put(path, nil)
}
