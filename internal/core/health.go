package core

import (
	"fmt"
	"net/http"
	"time"
)

// WaitForReady polls the mihomo API until it responds or timeout.
func WaitForReady(extCtl string, secret string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://%s/version", extCtl)

	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", url, nil)
		if secret != "" {
			req.Header.Set("Authorization", "Bearer "+secret)
		}

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("mihomo did not become ready within %s", timeout)
}
