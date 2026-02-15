package steam

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	vdf "github.com/andygrunwald/vdf"

	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// Source implements GameSource for Steam games
type Source struct {
	installPath string
	artCache    string
	config      Config
}

// Config holds Steam source configuration
type Config struct {
	InstallPath string // Override auto-detection
	APIKey      string // Steam Web API key
}

// Name returns the source identifier
func (s *Source) Name() string {
	return "steam"
}

// Init initializes the Steam source
func (s *Source) Init(config map[string]any) error {
	// Try to auto-detect Steam installation
	if config != nil {
		if path, ok := config["installPath"].(string); ok && path != "" {
			s.installPath = path
		}
		if apiKey, ok := config["apiKey"].(string); ok && apiKey != "" {
			s.config.APIKey = apiKey
		}
	}

	// Auto-detect if not configured
	if s.installPath == "" {
		path, err := s.detectSteamPath()
		if err != nil {
			return fmt.Errorf("failed to detect Steam installation: %w", err)
		}
		s.installPath = path
	}

	// Verify Steam exists
	if _, err := os.Stat(s.installPath); os.IsNotExist(err) {
		return fmt.Errorf("Steam not found at %s", s.installPath)
	}

	// Set up art cache
	s.artCache = filepath.Join(os.Getenv("HOME"), ".local", "share", "gentro", "cache", "steam", "art")
	if err := os.MkdirAll(s.artCache, 0755); err != nil {
		return fmt.Errorf("failed to create art cache path: %w", err)
	}

	return nil
}

// GetInstances returns all Steam games (installed + library)
// For now returns installed games only - Steam Web API integration planned
func (s *Source) GetInstances(ctx context.Context) ([]models.GameInstance, error) {
	return s.GetInstalledInstances(ctx)
}

// GetInstalledInstances returns only installed Steam games
func (s *Source) GetInstalledInstances(ctx context.Context) ([]models.GameInstance, error) {
	steamappsDir := filepath.Join(s.installPath, "steamapps")

	// Check if steamapps directory exists
	if _, err := os.Stat(steamappsDir); os.IsNotExist(err) {
		return []models.GameInstance{}, nil
	}

	// Find all appmanifest_*.acf files
	entries, err := os.ReadDir(steamappsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read steamapps directory: %w", err)
	}

	var instances []models.GameInstance
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isAppManifest(name) {
			continue
		}

		manifestPath := filepath.Join(steamappsDir, name)
		instance, err := ParseAppManifest(manifestPath)
		if err != nil {
			// Log error but continue processing other manifests
			fmt.Printf("Warning: failed to parse %s: %v\n", name, err)
			continue
		}

		// Update instance timestamps
		instance.UpdatedAt = time.Now()
		instances = append(instances, *instance)
	}

	return instances, nil
}

// isAppManifest checks if a filename is an appmanifest file
func isAppManifest(filename string) bool {
	return filepath.Ext(filename) == ".acf" && len(filename) > 12 && filename[:12] == "appmanifest_"
}

// Refresh updates Steam game data
func (s *Source) Refresh(ctx context.Context) error {
	// TODO: Re-fetch from Steam
	return nil
}

// GetGameArt returns Steam game art, fetching from CDN if not cached
func (s *Source) GetGameArt(ctx context.Context, instanceID string, artType string) ([]byte, string, error) {
	// Parse instance ID to get Steam App ID
	// instanceID format: "steam_{appid}"
	appID := strings.TrimPrefix(instanceID, "steam_")
	if appID == "" {
		return nil, "", fmt.Errorf("invalid instance ID format: %s", instanceID)
	}

	// Look for cached art
	artPath := filepath.Join(s.artCache, instanceID, artType+".jpg")

	// Check if art exists in cache
	data, err := os.ReadFile(artPath)
	if err == nil {
		return data, "image/jpeg", nil
	}

	// If not in cache and it's a "not found" error, fetch from Steam CDN
	if os.IsNotExist(err) {
		return s.fetchAndCacheArt(ctx, appID, artType, artPath)
	}

	// Some other error occurred reading the file
	return nil, "", fmt.Errorf("failed to read art: %w", err)
}

// fetchAndCacheArt downloads art from Steam CDN and caches it
func (s *Source) fetchAndCacheArt(ctx context.Context, appID, artType, artPath string) ([]byte, string, error) {
	// Build Steam CDN URL based on art type
	var cdnURL string
	switch artType {
	case "header":
		cdnURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", appID)
	case "library":
		cdnURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_600x900.jpg", appID)
	case "hero":
		cdnURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_hero.jpg", appID)
	case "logo":
		cdnURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/logo.png", appID)
	case "icon":
		cdnURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/icon.jpg", appID)
	default:
		// Default to header
		cdnURL = fmt.Sprintf("https://cdn.cloudflare.steamstatic.com/steam/apps/%s/header.jpg", appID)
	}

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Make the request
	req, err := http.NewRequestWithContext(ctx, "GET", cdnURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch art from Steam: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Steam CDN returned status %d for %s", resp.StatusCode, artType)
	}

	// Read the image data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read art data: %w", err)
	}

	// Create cache directory
	artDir := filepath.Dir(artPath)
	if err := os.MkdirAll(artDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create art cache directory: %w", err)
	}

	// Save to cache
	if err := os.WriteFile(artPath, data, 0644); err != nil {
		// Log but don't fail - we can still serve the image even if caching fails
		fmt.Printf("Warning: failed to cache art to %s: %v\n", artPath, err)
	}

	return data, "image/jpeg", nil
}

// detectSteamPath auto-detects Steam installation path
func (s *Source) detectSteamPath() (string, error) {
	switch runtime.GOOS {
	case "linux":
		return s.detectSteamLinux()
	case "windows":
		return s.detectSteamWindows()
	case "darwin":
		return s.detectSteamMac()
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// detectSteamLinux finds Steam on Linux
func (s *Source) detectSteamLinux() (string, error) {
	// Common Linux Steam paths
	candidates := []string{
		filepath.Join(os.Getenv("HOME"), ".local", "share", "Steam"),
		filepath.Join(os.Getenv("HOME"), ".steam", "steam"),
		filepath.Join(os.Getenv("HOME"), ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"), // Flatpak
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("Steam not found on Linux")
}

// detectSteamWindows finds Steam on Windows
func (s *Source) detectSteamWindows() (string, error) {
	// TODO: Implement Windows registry lookup
	return "", fmt.Errorf("Windows Steam detection not yet implemented")
}

// detectSteamMac finds Steam on macOS
func (s *Source) detectSteamMac() (string, error) {
	// Common macOS Steam path
	path := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Steam")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("Steam not found on macOS")
}

// ParseAppManifest parses a Steam appmanifest_*.acf file
func ParseAppManifest(path string) (*models.GameInstance, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open appmanifest: %w", err)
	}
	defer f.Close()

	p := vdf.NewParser(f)
	m, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse VDF: %w", err)
	}

	// VDF files have a root node (usually "AppState")
	var appState map[string]any
	for _, value := range m {
		if v, ok := value.(map[string]any); ok {
			appState = v
			break
		}
	}

	if appState == nil {
		return nil, fmt.Errorf("no AppState found in manifest")
	}

	// Extract basic fields
	appID := getString(appState, "appid")
	name := getString(appState, "name")
	installDir := getString(appState, "installdir")

	if appID == "" {
		return nil, fmt.Errorf("no appid found in manifest")
	}

	// Build source data from all VDF fields, with game name as displayName
	sourceData := make(map[string]any)
	for k, v := range appState {
		sourceData[k] = v
	}
	// Ensure displayName is set for getDisplayName lookup
	if name != "" {
		sourceData["displayName"] = name
	}

	// Build install path
	// path is like: /path/to/steam/steamapps/appmanifest_*.acf
	// We want: /path/to/steam/steamapps/common/<installDir>
	steamDir := filepath.Dir(path) // Gets the steamapps directory
	installPath := filepath.Join(steamDir, "common", installDir)

	// Extract file size if available
	fileSize := getInt64(appState, "SizeOnDisk")

	instance := &models.GameInstance{
		ID:          fmt.Sprintf("steam_%s", appID),
		GameID:      appID,
		Source:      "steam",
		Platform:    "steam",
		SourceID:    appID,
		Filename:    installDir,
		FileSize:    fileSize,
		Installed:   true,
		InstallPath: installPath,
		SourceData:  sourceData,
	}

	return instance, nil
}

// getString extracts a string value from a map[string]any, handling nested maps
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case map[string]any:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// getInt64 extracts an int64 value from a map[string]any (VDF stores numbers as strings)
func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			if i, err := strconv.ParseInt(val, 10, 64); err == nil {
				return i
			}
		case int64:
			return val
		case int:
			return int64(val)
		}
	}
	return 0
}

// Launch initiates the game via Steam URL protocol
func (s *Source) Launch(ctx context.Context, instance models.GameInstance) (*exec.Cmd, error) {
	// Extract AppID from SourceID
	appID := instance.SourceID
	if appID == "" {
		return nil, fmt.Errorf("no source ID for Steam instance")
	}

	// Build Steam URL
	url := fmt.Sprintf("steam://rungameid/%s", appID)

	// Open URL with platform-specific command
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to launch Steam URL: %w", err)
	}

	return cmd, nil
}

// MonitorProcess watches the Steam game process and emits status events
// For Steam, we use activity-based polling since Steam manages the actual game process
// The GamesService.monitorGameProcess handles the actual monitoring via isProcessRunningInPath
func (s *Source) MonitorProcess(ctx context.Context, instance models.GameInstance, cmd *exec.Cmd) {
	// Steam launches are indirect - the cmd here is just the URL opener (xdg-open, etc.)
	// The actual game process is managed by Steam, so we rely on activity-based detection
	// in GamesService.monitorGameProcess which polls for processes in the install path
}
