package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config is the data stored in ~/.claude-profiles/config.json.
type Config struct {
	ActiveProfile string   `json:"activeProfile"`
	Profiles      []string `json:"profiles"`
	KnownSkills   []string `json:"knownSkills"`
}

const configFile = "config.json"

// Load reads config.json from root. Returns empty Config (not an error) when
// the file does not exist yet — that means a fresh install.
func Load(root string) (Config, error) {
	path := filepath.Join(root, configFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes cfg to config.json in root atomically (write-then-rename).
func Save(root string, cfg Config) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(root, configFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(root, configFile))
}

// ActiveProfile returns the name of the currently active profile, or an empty
// string when no profile has been initialised yet.
func ActiveProfile(cfg Config) string {
	return cfg.ActiveProfile
}

// SetActive returns a new Config with ActiveProfile set to name.
// Callers must pass the result to Save to persist the change.
func SetActive(cfg Config, name string) Config {
	cfg.ActiveProfile = name
	return cfg
}
