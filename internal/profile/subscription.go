package profile

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// FetchSubscription downloads a profile from a remote URL and saves it to disk.
func FetchSubscription(url, savePath string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "mihomo-cli/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscription URL returned status %d", resp.StatusCode)
	}

	// Ensure directory exists
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create profile directory: %w", err)
	}

	// Write to temp file first, then rename for atomicity
	tmpPath := savePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("write profile data: %w", err)
	}

	if err := os.Rename(tmpPath, savePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("save profile: %w", err)
	}

	return nil
}
