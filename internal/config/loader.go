package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ErrConfigNotFound is returned when the config file doesn't exist
var ErrConfigNotFound = errors.New("config file not found")

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "familiar-says", "config.json"), nil
}

// Load loads the configuration from the default location
// Returns nil (not error) if config file doesn't exist
func Load() (*Config, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFromPath(configPath)
}

// LoadFromPath loads configuration from a specific path
// Returns nil (not error) if config file doesn't exist
func LoadFromPath(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil // Config file is optional, return nil without error
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
