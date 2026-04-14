// Package bootstrap handles the pre-TUI setup: download subscription,
// write mihomo config, launch mihomo if needed, and verify the API is up.
package bootstrap

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/boomyao/clash-cli/internal/api"
	"github.com/boomyao/clash-cli/internal/config"
	"github.com/boomyao/clash-cli/internal/core"
	"github.com/boomyao/clash-cli/internal/profile"
)

// Result describes what bootstrap accomplished.
type Result struct {
	// Manager is non-nil when bootstrap launched mihomo itself
	// (and is therefore responsible for stopping it on exit).
	Manager *core.Manager
	// SubscriptionURL is the URL that was applied this run (empty if none).
	SubscriptionURL string
}

// Run prepares the runtime: ensures mihomo has a valid config, ensures
// it is running, and returns a result describing ownership.
//
// Behavior:
//   - If urlArg is set, downloads it as the active subscription.
//   - If mihomo's API is reachable, leaves it alone (returns Manager == nil).
//   - Otherwise launches mihomo as a child process and waits for it to be ready.
func Run(cfg *config.AppConfig, urlArg string) (*Result, error) {
	mihomoDir := cfg.Core.ConfigDir
	if mihomoDir == "" {
		mihomoDir = config.DefaultMihomoConfigDir()
		cfg.Core.ConfigDir = mihomoDir
	}

	if err := os.MkdirAll(mihomoDir, 0755); err != nil {
		return nil, fmt.Errorf("create mihomo config dir: %w", err)
	}

	// Step 1: Apply any subscription URL passed on the CLI.
	if urlArg != "" {
		if err := installSubscription(cfg, urlArg); err != nil {
			return nil, fmt.Errorf("install subscription: %w", err)
		}
	}

	// Step 2: Ensure ~/.config/mihomo/config.yaml exists.
	// Prefer the active profile from cfg if one is set.
	mihomoConfigPath := filepath.Join(mihomoDir, "config.yaml")
	if _, err := os.Stat(mihomoConfigPath); os.IsNotExist(err) {
		if active := findActiveProfilePath(cfg); active != "" {
			if err := copyAndPatch(active, mihomoConfigPath); err != nil {
				return nil, fmt.Errorf("install profile as mihomo config: %w", err)
			}
		} else {
			if err := writeMinimalConfig(mihomoConfigPath); err != nil {
				return nil, fmt.Errorf("write minimal config: %w", err)
			}
		}
	}

	// Persist any cfg changes made above
	_ = cfg.Save()

	apiClient := api.NewClient(cfg.API.ExternalController, cfg.API.Secret)
	res := &Result{SubscriptionURL: urlArg}

	// Step 3: If mihomo is already running, just connect.
	if isAPIAlive(apiClient, 1*time.Second) {
		return res, nil
	}

	// Step 4: Launch mihomo ourselves.
	if !cfg.Core.AutoStart {
		return nil, errors.New("mihomo is not running and auto_start is disabled in config")
	}

	binPath, err := core.FindBinary(cfg.Core.BinaryPath)
	if err != nil {
		// Fall back to PATH lookup (handles stale or invalid configured paths)
		if cfg.Core.BinaryPath != "" && cfg.Core.BinaryPath != "mihomo" {
			binPath, err = core.FindBinary("")
			if err == nil {
				cfg.Core.BinaryPath = binPath
				_ = cfg.Save()
			}
		}
		if err != nil {
			return nil, fmt.Errorf("locate mihomo binary: %w (try --mihomo-bin)", err)
		}
	}

	logPath := filepath.Join(config.DataDir(), "mihomo.log")
	mgr := core.NewManager(binPath, mihomoDir, cfg.API.ExternalController, logPath)
	if err := mgr.Start(); err != nil {
		return nil, fmt.Errorf("start mihomo: %w", err)
	}

	if err := core.WaitForReady(cfg.API.ExternalController, cfg.API.Secret, 10*time.Second); err != nil {
		_ = mgr.Stop()
		return nil, fmt.Errorf("mihomo did not become ready: %w", err)
	}

	res.Manager = mgr
	return res, nil
}

// installSubscription downloads urlArg, writes it to mihomo's config dir
// (with a safety patch), records it as a profile, and marks it active.
func installSubscription(cfg *config.AppConfig, url string) error {
	mihomoDir := cfg.Core.ConfigDir
	mihomoConfigPath := filepath.Join(mihomoDir, "config.yaml")

	// Reuse existing profile if URL matches one already stored
	for _, p := range cfg.Profiles.Items {
		if p.URL == url {
			cfg.Profiles.Active = p.ID
			// Re-fetch latest content to mihomo dir
			profPath := filepath.Join(config.DataDir(), p.Path)
			if err := profile.FetchSubscription(url, profPath); err != nil {
				return err
			}
			return copyAndPatch(profPath, mihomoConfigPath)
		}
	}

	// New subscription — register and download
	mgr := profile.NewManager(cfg)
	prof, err := mgr.Add("Default Subscription", "remote", url)
	if err != nil {
		return err
	}
	cfg.Profiles.Active = prof.ID

	src := filepath.Join(config.DataDir(), prof.Path)
	return copyAndPatch(src, mihomoConfigPath)
}

// findActiveProfilePath returns the on-disk path of the active profile yaml,
// or "" if there is none.
func findActiveProfilePath(cfg *config.AppConfig) string {
	if cfg.Profiles.Active == "" {
		return ""
	}
	for _, p := range cfg.Profiles.Items {
		if p.ID == cfg.Profiles.Active {
			return filepath.Join(config.DataDir(), p.Path)
		}
	}
	return ""
}

// copyAndPatch reads src, applies safety patches, and writes to dst atomically.
func copyAndPatch(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	patched := applySafetyPatches(data)

	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, patched, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// applySafetyPatches rewrites parts of a mihomo config that would otherwise
// require root or break in our non-privileged context.
//
//   - dns.listen: ":53" / "0.0.0.0:53" → "0.0.0.0:1053" (avoids needing root)
func applySafetyPatches(data []byte) []byte {
	// Match lines like `  listen: 0.0.0.0:53` or `  listen: :53` (any leading space)
	re := regexp.MustCompile(`(?m)^(\s*listen:\s*)(0\.0\.0\.0:53|:53|127\.0\.0\.1:53)\s*$`)
	return re.ReplaceAll(data, []byte("${1}0.0.0.0:1053"))
}

// writeMinimalConfig creates an empty-shell mihomo config when no profile exists.
func writeMinimalConfig(path string) error {
	const minimal = `mixed-port: 7890
allow-lan: false
mode: rule
log-level: info
external-controller: 127.0.0.1:9090
secret: ""
proxies: []
proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - DIRECT
rules:
  - MATCH,PROXY
`
	return os.WriteFile(path, []byte(minimal), 0644)
}

// isAPIAlive probes /version with a short timeout.
func isAPIAlive(client *api.Client, timeout time.Duration) bool {
	hc := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET", client.BaseURL()+"/version", nil)
	if err != nil {
		return false
	}
	if s := client.Secret(); s != "" {
		req.Header.Set("Authorization", "Bearer "+s)
	}
	resp, err := hc.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == 200
}

// LooksLikeURL returns true if s is an http/https URL.
func LooksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
