package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/boomyao/clash-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// Manager handles profile CRUD operations.
type Manager struct {
	cfg         *config.AppConfig
	profilesDir string
}

// NewManager creates a profile manager.
func NewManager(cfg *config.AppConfig) *Manager {
	return &Manager{
		cfg:         cfg,
		profilesDir: config.ProfilesDir(),
	}
}

// List returns all profiles.
func (m *Manager) List() []Profile {
	profiles := make([]Profile, len(m.cfg.Profiles.Items))
	for i, item := range m.cfg.Profiles.Items {
		profiles[i] = Profile{
			ID:                 item.ID,
			Name:               item.Name,
			Type:               item.Type,
			URL:                item.URL,
			Path:               item.Path,
			AutoUpdateInterval: item.AutoUpdateInterval,
		}
		if item.UpdatedAt != "" {
			profiles[i].UpdatedAt, _ = time.Parse(time.RFC3339, item.UpdatedAt)
		}
	}
	return profiles
}

// ActiveID returns the currently active profile ID.
func (m *Manager) ActiveID() string {
	return m.cfg.Profiles.Active
}

// ActiveProfile returns the active profile, or nil if none.
func (m *Manager) ActiveProfile() *Profile {
	for _, p := range m.List() {
		if p.ID == m.cfg.Profiles.Active {
			return &p
		}
	}
	return nil
}

// Add creates a new profile.
func (m *Manager) Add(name, profileType, url string) (*Profile, error) {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	filename := id + ".yaml"
	savePath := filepath.Join(m.profilesDir, filename)

	if profileType == "remote" && url != "" {
		if err := FetchSubscription(url, savePath); err != nil {
			return nil, fmt.Errorf("fetch subscription: %w", err)
		}
	} else {
		// Create an empty local config
		if err := os.MkdirAll(m.profilesDir, 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(savePath, []byte("# mihomo config\n"), 0644); err != nil {
			return nil, err
		}
	}

	profile := Profile{
		ID:        id,
		Name:      name,
		Type:      profileType,
		URL:       url,
		Path:      filepath.Join("profiles", filename),
		UpdatedAt: time.Now(),
	}

	m.cfg.Profiles.Items = append(m.cfg.Profiles.Items, config.ProfileConfig{
		ID:        profile.ID,
		Name:      profile.Name,
		Type:      profile.Type,
		URL:       profile.URL,
		Path:      profile.Path,
		UpdatedAt: profile.UpdatedAt.Format(time.RFC3339),
	})

	if err := m.cfg.Save(); err != nil {
		return nil, err
	}

	return &profile, nil
}

// Update re-fetches a remote profile from its subscription URL.
func (m *Manager) Update(id string) error {
	for i, item := range m.cfg.Profiles.Items {
		if item.ID == id {
			if item.Type != "remote" || item.URL == "" {
				return fmt.Errorf("profile %s is not a remote subscription", id)
			}

			savePath := filepath.Join(config.DataDir(), item.Path)
			if err := FetchSubscription(item.URL, savePath); err != nil {
				return err
			}

			m.cfg.Profiles.Items[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return m.cfg.Save()
		}
	}
	return fmt.Errorf("profile %s not found", id)
}

// Delete removes a profile.
func (m *Manager) Delete(id string) error {
	for i, item := range m.cfg.Profiles.Items {
		if item.ID == id {
			// Remove the file
			savePath := filepath.Join(config.DataDir(), item.Path)
			os.Remove(savePath)

			// Remove from config
			m.cfg.Profiles.Items = append(m.cfg.Profiles.Items[:i], m.cfg.Profiles.Items[i+1:]...)

			// Clear active if it was the deleted profile
			if m.cfg.Profiles.Active == id {
				m.cfg.Profiles.Active = ""
			}

			return m.cfg.Save()
		}
	}
	return fmt.Errorf("profile %s not found", id)
}

// SetActive sets the active profile.
func (m *Manager) SetActive(id string) error {
	// Verify the profile exists
	found := false
	for _, item := range m.cfg.Profiles.Items {
		if item.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("profile %s not found", id)
	}

	m.cfg.Profiles.Active = id
	return m.cfg.Save()
}

// GetProfilePath returns the absolute file path for a profile.
func (m *Manager) GetProfilePath(id string) (string, error) {
	for _, item := range m.cfg.Profiles.Items {
		if item.ID == id {
			return filepath.Join(config.DataDir(), item.Path), nil
		}
	}
	return "", fmt.Errorf("profile %s not found", id)
}

// ValidateProfile checks if a profile YAML is valid.
func (m *Manager) ValidateProfile(id string) error {
	path, err := m.GetProfilePath(id)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read profile: %w", err)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	return nil
}
