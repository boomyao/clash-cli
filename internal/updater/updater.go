// Package updater handles version checks and self-update against the
// boomyao/clash-cli GitHub releases.
package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	repo       = "boomyao/clash-cli"
	binName    = "clashc"
	httpAgent  = "clashc-updater"
)

// Release is a minimal subset of GitHub's release JSON payload.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a single release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// LatestRelease fetches the latest release metadata from GitHub.
// The HTTP timeout is short — callers shouldn't block startup on it.
func LatestRelease(timeout time.Duration) (*Release, error) {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("GET",
		"https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", httpAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// IsNewer returns true if `latest` is strictly greater than `current`.
// Both are expected to look like "v1.2.3" (the leading "v" is optional).
// Non-numeric or "dev" current versions count as older than any release.
func IsNewer(current, latest string) bool {
	cParts, cOK := parseVersion(current)
	lParts, lOK := parseVersion(latest)
	if !lOK {
		return false
	}
	if !cOK {
		return true // dev / commit-hash / blank → assume update available
	}
	for i := 0; i < 3; i++ {
		if lParts[i] > cParts[i] {
			return true
		}
		if lParts[i] < cParts[i] {
			return false
		}
	}
	return false
}

// parseVersion turns "v1.2.3" / "1.2.3" into [1,2,3]. Returns ok=false
// for anything that doesn't look like semver-major-minor-patch.
func parseVersion(v string) ([3]int, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// Allow trailing "-snapshot" / "-rc1" — strip the suffix
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

// Run downloads the latest release for the current platform and replaces
// the running clashc binary in place. Returns the new version string on
// success. The caller must restart clashc to pick up the new binary.
//
// w is used for human-readable progress output (typically os.Stderr).
func Run(currentVersion string, w io.Writer) (string, error) {
	fmt.Fprintln(w, "▶ Checking latest release...")
	rel, err := LatestRelease(15 * time.Second)
	if err != nil {
		return "", fmt.Errorf("query release: %w", err)
	}
	fmt.Fprintf(w, "  current: %s\n", currentVersion)
	fmt.Fprintf(w, "  latest:  %s\n", rel.TagName)

	if !IsNewer(currentVersion, rel.TagName) {
		fmt.Fprintln(w, "✓ Already up to date.")
		return rel.TagName, nil
	}

	// Locate the binary asset for this OS/arch
	verNoV := strings.TrimPrefix(rel.TagName, "v")
	wantName := fmt.Sprintf("clashc_%s_%s_%s.tar.gz", verNoV, runtime.GOOS, runtime.GOARCH)
	var asset *Asset
	for i := range rel.Assets {
		if rel.Assets[i].Name == wantName {
			asset = &rel.Assets[i]
			break
		}
	}
	if asset == nil {
		return "", fmt.Errorf("no asset named %s in release %s", wantName, rel.TagName)
	}

	fmt.Fprintf(w, "▶ Downloading %s...\n", asset.Name)
	tmpDir, err := os.MkdirTemp("", "clashc-update-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	tmpBin, err := downloadAndExtract(asset.BrowserDownloadURL, tmpDir)
	if err != nil {
		return "", err
	}

	// Replace the running binary atomically. On Unix you can rename(2) over
	// a currently-executing file; the running process keeps using the old
	// inode, the new launches use the new file.
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate own executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = resolved
	}

	if err := replaceBinary(tmpBin, exePath); err != nil {
		return "", fmt.Errorf("replace %s: %w (try running as root if installed system-wide)", exePath, err)
	}

	fmt.Fprintf(w, "✓ Updated %s → %s\n", currentVersion, rel.TagName)
	fmt.Fprintf(w, "  Installed at: %s\n", exePath)
	fmt.Fprintln(w, "  Restart clashc to use the new version.")
	return rel.TagName, nil
}

// downloadAndExtract fetches a tar.gz, finds clashc inside, and writes it
// to a temp file. Returns the path to the extracted binary.
func downloadAndExtract(url, tmpDir string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", httpAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) != binName {
			continue
		}
		out := filepath.Join(tmpDir, binName)
		f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
		return out, nil
	}
	return "", fmt.Errorf("clashc binary not found in archive")
}

// replaceBinary writes src over dst. On most filesystems we can just
// rename, but cross-device or read-only-mount failures fall back to copy.
func replaceBinary(src, dst string) error {
	// Same dir → safe rename
	if filepath.Dir(src) == filepath.Dir(dst) {
		return os.Rename(src, dst)
	}
	// Try rename across dirs (works on same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Fallback: copy via temp file in same dir, then rename
	tmp := dst + ".update.tmp"
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	out.Close()
	return os.Rename(tmp, dst)
}
