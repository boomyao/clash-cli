package api

// GetRules fetches all routing rules.
func (c *Client) GetRules() (*RulesResponse, error) {
	var result RulesResponse
	if err := c.get("/rules", &result); err != nil {
		return nil, err
	}
	return &result, nil
}
