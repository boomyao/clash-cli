package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// FindBinary locates the mihomo binary. Checks the given path first,
// then falls back to searching PATH.
func FindBinary(path string) (string, error) {
	if path != "" {
		// Check if the specified path exists and is executable
		absPath, err := exec.LookPath(path)
		if err != nil {
			return "", fmt.Errorf("mihomo binary not found at %s: %w", path, err)
		}
		return absPath, nil
	}

	// Try common names
	for _, name := range []string{"mihomo", "clash-meta", "clash"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("mihomo binary not found in PATH; specify with --mihomo-bin")
}

// GetVersion runs mihomo -v and parses the version string.
func GetVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get mihomo version: %w", err)
	}

	// Output is typically: "Mihomo Meta v1.18.x ..."
	line := strings.TrimSpace(string(output))
	return line, nil
}
