package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)


// Manager handles loading and saving application configuration
type Manager struct {
	path string
	data *Config
	mu   sync.RWMutex
}

// Config represents the application configuration
type Config struct {
	// Filters contains user filter preferences
	Filters FilterConfig `toml:"filters"`
}

// FilterConfig contains filter-related settings
type FilterConfig struct {
	// Steam contains Steam-specific filter settings
	Steam SteamFilterConfig `toml:"steam"`
}

// SteamFilterConfig contains Steam-specific filter settings
type SteamFilterConfig struct {
	// ExcludeTools controls whether Steam tools are hidden
	ExcludeTools bool `toml:"excludeTools"`
}

var defaultConfig = Config{
	Filters: FilterConfig{
		Steam: SteamFilterConfig{
			ExcludeTools:  true,
		},
	},
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	manager := &Manager{
		path: configPath,
		data: &defaultConfig,
	}

	// Try to load existing config
	if err := manager.Load(); err != nil {
		// If file doesn't exist, save defaults
		if os.IsNotExist(err) {
			if err := manager.Save(); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	return manager, nil
}

// Load reads configuration from disk
func (m *Manager) Load() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, err := toml.DecodeFile(m.path, m.data); err != nil {
		return err
	}

	return nil
}

// Save writes configuration to disk
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.RUnlock()

	file, err := os.Create(m.path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(m.data); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.data
}

// SetFilters updates filter configuration
func (m *Manager) SetFilters(filters FilterConfig) error {
	m.mu.Lock()
	m.data.Filters = filters
	m.mu.Unlock()

	return m.Save()
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".local", "share", "gentro", "config", "gentro.toml")
}
