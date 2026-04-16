package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SuperadminEntry struct {
	Email       string `yaml:"email"`
	DisplayName string `yaml:"display_name"`
	DevPassword string `yaml:"dev_password,omitempty"`
}

type SuperadminsConfig struct {
	Superadmins []SuperadminEntry `yaml:"superadmins"`
}

// LoadSuperadmins parses the superadmins.yaml file and returns a map
// keyed by email address. Superadmins are held in memory (not in users table)
// and checked at the API layer for elevated privileges.
func LoadSuperadmins(path string) (map[string]SuperadminEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read superadmins config: %w", err)
	}
	var cfg SuperadminsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse superadmins config: %w", err)
	}
	m := make(map[string]SuperadminEntry, len(cfg.Superadmins))
	for _, s := range cfg.Superadmins {
		m[s.Email] = s
	}
	return m, nil
}
