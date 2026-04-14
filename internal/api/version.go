package api

// GetVersion fetches the mihomo version information.
func (c *Client) GetVersion() (*VersionResponse, error) {
	var result VersionResponse
	if err := c.get("/version", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Restart restarts the mihomo core.
func (c *Client) Restart() error {
	return c.post("/restart", nil)
}
