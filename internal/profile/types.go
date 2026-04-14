package profile

import "time"

// Profile represents a mihomo configuration profile.
type Profile struct {
	ID                 string    `yaml:"id"`
	Name               string    `yaml:"name"`
	Type               string    `yaml:"type"` // "remote" or "local"
	URL                string    `yaml:"url,omitempty"`
	Path               string    `yaml:"path"`
	UpdatedAt          time.Time `yaml:"updated_at,omitempty"`
	AutoUpdateInterval int       `yaml:"auto_update_interval,omitempty"` // seconds
}

// IsRemote returns true if this profile is fetched from a URL.
func (p *Profile) IsRemote() bool {
	return p.Type == "remote"
}

// NeedsUpdate returns true if the profile is due for an update.
func (p *Profile) NeedsUpdate() bool {
	if !p.IsRemote() || p.AutoUpdateInterval <= 0 {
		return false
	}
	return time.Since(p.UpdatedAt) > time.Duration(p.AutoUpdateInterval)*time.Second
}
