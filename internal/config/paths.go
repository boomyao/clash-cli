package config

import (
	"os"
	"path/filepath"
)

const appName = "clash-cli"

// ConfigDir returns the configuration directory path.
// Uses $XDG_CONFIG_HOME/clash-cli or defaults to ~/.config/clash-cli
func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName)
}

// DataDir returns the data directory path.
// Uses $XDG_DATA_HOME/clash-cli or defaults to ~/.local/share/clash-cli
func DataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", appName)
}

// ProfilesDir returns the directory where profile YAML files are stored.
func ProfilesDir() string {
	return filepath.Join(DataDir(), "profiles")
}

// DefaultMihomoConfigDir returns the default mihomo config directory.
func DefaultMihomoConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "mihomo")
}

// ConfigFilePath returns the path to the main app config file.
func ConfigFilePath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// EnsureDirs creates all necessary directories if they don't exist.
// Also migrates legacy ~/.config/mihomo-cli + ~/.local/share/mihomo-cli
// directories on first run after the rename.
func EnsureDirs() error {
	migrateLegacyDirs()

	dirs := []string{
		ConfigDir(),
		DataDir(),
		ProfilesDir(),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// migrateLegacyDirs renames any pre-rename directories so existing users
// don't lose their profiles/config when upgrading from mihomo-cli to clash-cli.
// Best-effort: silently ignored on error or if the new dir already exists.
func migrateLegacyDirs() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	pairs := [][2]string{
		{filepath.Join(home, ".config", "mihomo-cli"), filepath.Join(home, ".config", "clash-cli")},
		{filepath.Join(home, ".local", "share", "mihomo-cli"), filepath.Join(home, ".local", "share", "clash-cli")},
	}
	for _, p := range pairs {
		oldDir, newDir := p[0], p[1]
		if _, err := os.Stat(oldDir); err != nil {
			continue
		}
		if _, err := os.Stat(newDir); err == nil {
			continue // new dir already exists, skip
		}
		_ = os.Rename(oldDir, newDir)
	}
}
