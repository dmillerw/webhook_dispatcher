package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

// Loader manages configuration loading and hot-reload
type Loader struct {
	configPath string
	config     atomic.Value // stores *Config
}

// NewLoader creates a new config loader
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// Load loads the configuration from disk
func (l *Loader) Load() error {
	config, err := loadFromFile(l.configPath)
	if err != nil {
		return err
	}

	// Validate before storing
	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	l.config.Store(config)
	return nil
}

// Reload reloads the configuration from disk
func (l *Loader) Reload() error {
	newConfig, err := loadFromFile(l.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate before swapping
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Atomic swap
	l.config.Store(newConfig)
	return nil
}

// GetConfig returns the current configuration (thread-safe)
func (l *Loader) GetConfig() *Config {
	if cfg := l.config.Load(); cfg != nil {
		return cfg.(*Config)
	}
	return &Config{}
}

// loadFromFile loads configuration from a JSON file
func loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to disk
func (l *Loader) SaveConfig(config *Config) error {
	// Validate before saving
	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(l.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Update in-memory config
	l.config.Store(config)
	return nil
}
