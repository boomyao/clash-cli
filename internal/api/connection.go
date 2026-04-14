package api

import "fmt"

// GetConnections fetches all active connections.
func (c *Client) GetConnections() (*ConnectionsResponse, error) {
	var result ConnectionsResponse
	if err := c.get("/connections", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CloseAllConnections closes all active connections.
func (c *Client) CloseAllConnections() error {
	return c.delete("/connections")
}

// CloseConnection closes a specific connection by ID.
func (c *Client) CloseConnection(id string) error {
	return c.delete(fmt.Sprintf("/connections/%s", id))
}
