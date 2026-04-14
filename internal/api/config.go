package api

import "fmt"

// GetConfig fetches the current mihomo configuration.
func (c *Client) GetConfig() (*ConfigResponse, error) {
	var result ConfigResponse
	if err := c.get("/configs", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PatchConfig updates specific configuration fields.
func (c *Client) PatchConfig(patch ConfigPatch) error {
	return c.patch("/configs", patch)
}

// ReloadConfig forces mihomo to reload its configuration from a file path.
func (c *Client) ReloadConfig(path string, force bool) error {
	url := "/configs"
	if force {
		url += "?force=true"
	}
	body := map[string]string{"path": path}
	return c.put(url, body)
}

// SetMode changes the clash running mode (rule/global/direct).
func (c *Client) SetMode(mode string) error {
	return c.PatchConfig(ConfigPatch{"mode": mode})
}

// SetAllowLan toggles the allow-lan setting.
func (c *Client) SetAllowLan(allow bool) error {
	return c.PatchConfig(ConfigPatch{"allow-lan": allow})
}

// SetTunEnabled toggles the TUN mode.
func (c *Client) SetTunEnabled(enabled bool) error {
	return c.PatchConfig(ConfigPatch{
		"tun": map[string]interface{}{
			"enable": enabled,
		},
	})
}

// SetLogLevel changes the log level.
func (c *Client) SetLogLevel(level string) error {
	return c.PatchConfig(ConfigPatch{"log-level": level})
}

// UpdateGeoData triggers a GeoIP/GeoSite database update.
func (c *Client) UpdateGeoData() error {
	return c.post("/configs/geo", nil)
}

// FlushFakeIP flushes the FakeIP cache.
func (c *Client) FlushFakeIP() error {
	return c.post("/cache/fakeip/flush", nil)
}

// FlushDNS flushes the DNS cache.
func (c *Client) FlushDNS() error {
	return c.post("/cache/dns/flush", nil)
}

// DNSQuery performs a DNS lookup through mihomo.
func (c *Client) DNSQuery(name, qtype string) (map[string]interface{}, error) {
	var result map[string]interface{}
	path := fmt.Sprintf("/dns/query?name=%s&type=%s", name, qtype)
	if err := c.get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}
