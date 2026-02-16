package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")

	// Test creating new manager
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Verify defaults
	cfg := manager.Get()
	if !cfg.Filters.Steam.ExcludeTools {
		t.Error("Expected ExcludeTools to default to true")
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestLoadAndSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.toml")

	// Create manager with defaults
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Modify config
	newFilters := FilterConfig{
		Steam: SteamFilterConfig{
			ExcludeTools: false,
		},
	}

	if err := manager.SetFilters(newFilters); err != nil {
		t.Fatalf("Failed to set filters: %v", err)
	}

	// Create new manager and load saved config
	manager2, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}

	cfg := manager2.Get()
	if cfg.Filters.Steam.ExcludeTools {
		t.Error("Expected ExcludeTools to be false after save/load")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Error("DefaultConfigPath returned empty string")
	}
	if !contains(path, ".local/share/gentro/config") {
		t.Errorf("Expected path to contain '.local/share/gentro/config', got: %s", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
