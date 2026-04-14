package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig is the root configuration for mihomo-cli itself.
type AppConfig struct {
	Core        CoreConfig        `yaml:"core"`
	API         APIConfig         `yaml:"api"`
	SystemProxy SystemProxyConfig `yaml:"system_proxy"`
	Profiles    ProfilesConfig    `yaml:"profiles"`
	UI          UIConfig          `yaml:"ui"`
}

// CoreConfig configures the mihomo binary and process management.
type CoreConfig struct {
	BinaryPath  string `yaml:"binary_path"`
	ConfigDir   string `yaml:"config_dir"`
	AutoStart   bool   `yaml:"auto_start"`
	AutoRestart bool   `yaml:"auto_restart"`
}

// APIConfig configures the connection to mihomo's REST API.
type APIConfig struct {
	ExternalController string `yaml:"external_controller"`
	Secret             string `yaml:"secret"`
}

// SystemProxyConfig configures system proxy behavior.
type SystemProxyConfig struct {
	EnableOnStart bool `yaml:"enable_on_start"`
	DisableOnExit bool `yaml:"disable_on_exit"`
}

// ProfilesConfig stores the list of profiles and the active one.
type ProfilesConfig struct {
	Active string          `yaml:"active"`
	Items  []ProfileConfig `yaml:"items"`
}

// ProfileConfig describes a single profile (subscription or local).
type ProfileConfig struct {
	ID                 string `yaml:"id"`
	Name               string `yaml:"name"`
	Type               string `yaml:"type"` // "remote" or "local"
	URL                string `yaml:"url,omitempty"`
	Path               string `yaml:"path"`
	UpdatedAt          string `yaml:"updated_at,omitempty"`
	AutoUpdateInterval int    `yaml:"auto_update_interval,omitempty"` // seconds, 0 = disabled
}

// UIConfig configures the TUI appearance and behavior.
type UIConfig struct {
	DefaultTab int    `yaml:"default_tab"`
	LogLevel   string `yaml:"log_level"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Core: CoreConfig{
			BinaryPath:  "mihomo",
			ConfigDir:   DefaultMihomoConfigDir(),
			AutoStart:   true,
			AutoRestart: true,
		},
		API: APIConfig{
			ExternalController: "127.0.0.1:9090",
			Secret:             "",
		},
		SystemProxy: SystemProxyConfig{
			EnableOnStart: false,
			DisableOnExit: true,
		},
		Profiles: ProfilesConfig{
			Active: "",
			Items:  []ProfileConfig{},
		},
		UI: UIConfig{
			DefaultTab: 0,
			LogLevel:   "info",
		},
	}
}

// Load reads the config file from disk, or returns defaults if it doesn't exist.
func Load() (*AppConfig, error) {
	cfg := DefaultConfig()
	path := ConfigFilePath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the config to disk.
func (c *AppConfig) Save() error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFilePath(), data, 0644)
}
