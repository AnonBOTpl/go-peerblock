package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Persistence handles reading and writing the config file.
type Persistence struct {
	filePath string
}

// NewPersistence creates a new config persistence layer.
// The config file is stored in %APPDATA%/go-peerblock/config.json.
func NewPersistence() *Persistence {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("HOME"), ".config")
	}
	dir := filepath.Join(appData, "go-peerblock")
	return &Persistence{
		filePath: filepath.Join(dir, "config.json"),
	}
}

// Load reads the config from disk. Returns defaults if the file doesn't exist.
func (p *Persistence) Load() (*Config, error) {
	data, err := os.ReadFile(p.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	cfg := Defaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to disk.
func (p *Persistence) Save(cfg *Config) error {
	dir := filepath.Dir(p.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}

	if err := os.WriteFile(p.filePath, data, 0644); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}
	return nil
}

// ConfigPath returns the full path to the config file.
func (p *Persistence) ConfigPath() string {
	return p.filePath
}
