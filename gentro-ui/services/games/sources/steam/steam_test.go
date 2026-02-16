package steam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsTool(t *testing.T) {
	// Create temporary directories for testing
	tempDir := t.TempDir()

	// Test case 1: Directory with toolmanifest.vdf (tool)
	toolDir := filepath.Join(tempDir, "Proton - Experimental")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	toolManifestPath := filepath.Join(toolDir, "toolmanifest.vdf")
	if err := os.WriteFile(toolManifestPath, []byte(`"manifest"\n{\n}`), 0644); err != nil {
		t.Fatalf("failed to create toolmanifest.vdf: %v", err)
	}

	if !isTool(toolDir) {
		t.Errorf("isTool(%s) = false, want true", toolDir)
	}

	// Test case 2: Directory without toolmanifest.vdf (game)
	gameDir := filepath.Join(tempDir, "The Witcher 3")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatalf("failed to create game dir: %v", err)
	}

	if isTool(gameDir) {
		t.Errorf("isTool(%s) = true, want false", gameDir)
	}

	// Test case 3: Non-existent directory
	nonExistentDir := filepath.Join(tempDir, "non-existent")
	if isTool(nonExistentDir) {
		t.Errorf("isTool(%s) = true, want false for non-existent dir", nonExistentDir)
	}
}

func TestGetSteamType(t *testing.T) {
	tempDir := t.TempDir()

	// Test case 1: Tool directory
	toolDir := filepath.Join(tempDir, "SteamLinuxRuntime_sniper")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("failed to create tool dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, "toolmanifest.vdf"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create toolmanifest.vdf: %v", err)
	}

	if got := getSteamType("12345", toolDir); got != "tool" {
		t.Errorf("getSteamType(%s) = %s, want tool", toolDir, got)
	}

	// Test case 2: Game directory
	gameDir := filepath.Join(tempDir, "Counter-Strike 2")
	if err := os.MkdirAll(gameDir, 0755); err != nil {
		t.Fatalf("failed to create game dir: %v", err)
	}

	if got := getSteamType("730", gameDir); got != "game" {
		t.Errorf("getSteamType(%s) = %s, want game", gameDir, got)
	}
}

func TestParseAppManifest_SetsSteamType(t *testing.T) {
	// This test would require a real appmanifest file
	// For now, we'll just verify the structure exists

	// Create a mock appmanifest for testing
	tempDir := t.TempDir()
	steamappsDir := filepath.Join(tempDir, "steamapps")
	commonDir := filepath.Join(steamappsDir, "common")

	// Create tool directory with manifest
	toolInstallDir := filepath.Join(commonDir, "Proton - Experimental")
	if err := os.MkdirAll(toolInstallDir, 0755); err != nil {
		t.Fatalf("failed to create tool install dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(toolInstallDir, "toolmanifest.vdf"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create toolmanifest.vdf: %v", err)
	}

	// Create mock appmanifest
	manifestContent := `"AppState"
{
	"appid"		"1493710"
	"name"		"Proton Experimental"
	"installdir"		"Proton - Experimental"
	"StateFlags"		"4"
	"LastUpdated"		"1769622076"
}`

	manifestPath := filepath.Join(steamappsDir, "appmanifest_1493710.acf")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to create appmanifest: %v", err)
	}

	// Parse the manifest
	instance, err := ParseAppManifest(manifestPath)
	if err != nil {
		t.Fatalf("ParseAppManifest failed: %v", err)
	}

	// Verify steam.type is set correctly
	if instance.CustomMetadata == nil {
		t.Fatal("CustomMetadata is nil")
	}

	steamType, ok := instance.CustomMetadata["steam.type"].(string)
	if !ok {
		t.Fatalf("steam.type not found in CustomMetadata: %v", instance.CustomMetadata)
	}

	if steamType != "tool" {
		t.Errorf("steam.type = %s, want tool", steamType)
	}

	// Now test with a game (no toolmanifest)
	gameInstallDir := filepath.Join(commonDir, "TestGame")
	if err := os.MkdirAll(gameInstallDir, 0755); err != nil {
		t.Fatalf("failed to create game install dir: %v", err)
	}

	gameManifestContent := `"AppState"
{
	"appid"		"12345"
	"name"		"Test Game"
	"installdir"		"TestGame"
	"StateFlags"		"4"
}`

	gameManifestPath := filepath.Join(steamappsDir, "appmanifest_12345.acf")
	if err := os.WriteFile(gameManifestPath, []byte(gameManifestContent), 0644); err != nil {
		t.Fatalf("failed to create game appmanifest: %v", err)
	}

	gameInstance, err := ParseAppManifest(gameManifestPath)
	if err != nil {
		t.Fatalf("ParseAppManifest failed for game: %v", err)
	}

	gameSteamType, ok := gameInstance.CustomMetadata["steam.type"].(string)
	if !ok {
		t.Fatalf("steam.type not found in game CustomMetadata: %v", gameInstance.CustomMetadata)
	}

	if gameSteamType != "game" {
		t.Errorf("steam.type = %s, want game", gameSteamType)
	}
}

// BenchmarkIsTool benchmarks the tool detection
func BenchmarkIsTool(b *testing.B) {
	tempDir := b.TempDir()
	toolDir := filepath.Join(tempDir, "tool")
	os.MkdirAll(toolDir, 0755)
	os.WriteFile(filepath.Join(toolDir, "toolmanifest.vdf"), []byte("test"), 0644)

	gameDir := filepath.Join(tempDir, "game")
	os.MkdirAll(gameDir, 0755)

	b.Run("Tool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isTool(toolDir)
		}
	})

	b.Run("Game", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isTool(gameDir)
		}
	})
}
