package emulated

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/rhythmerc/gentro-ui/services/games/emulator"
	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// Source implements GameSource for emulated games (ROMs)
type Source struct {
	config                    Config
	basePath                  string
	platforms                 map[string]PlatformConfig
	artCache                  string
	emuService                *emulator.Service
	logger                    *slog.Logger
	emulatorAvailabilityCache map[string]bool
}

// Config holds emulated source configuration
type Config struct {
	BasePath  string
	Platforms map[string]PlatformConfig
}

// PlatformConfig defines platform-specific settings
type PlatformConfig struct {
	Extensions  []string
	DisplayName string
	ArtTypes    []string
}

// Common ROM extensions by platform
var defaultPlatformConfigs = map[string]PlatformConfig{
	"wii": {
		Extensions:  []string{".wbfs", ".iso", ".ciso", ".gcz"},
		DisplayName: "Nintendo Wii",
		ArtTypes:    []string{"boxart", "screenshot"},
	},
	"gamecube": {
		Extensions:  []string{".iso", ".ciso", ".gcz"},
		DisplayName: "Nintendo GameCube",
		ArtTypes:    []string{"boxart", "screenshot"},
	},
	"snes": {
		Extensions:  []string{".sfc", ".smc", ".fig", ".zip"},
		DisplayName: "Super Nintendo",
		ArtTypes:    []string{"boxart", "screenshot"},
	},
	"nes": {
		Extensions:  []string{".nes", ".zip", ".7z"},
		DisplayName: "Nintendo Entertainment System",
		ArtTypes:    []string{"boxart", "screenshot"},
	},
	"n64": {
		Extensions:  []string{".z64", ".n64", ".v64"},
		DisplayName: "Nintendo 64",
		ArtTypes:    []string{"boxart", "screenshot"},
	},
	"ps2": {
		Extensions:  []string{".iso", ".bin", ".img"},
		DisplayName: "PlayStation 2",
		ArtTypes:    []string{"boxart", "screenshot"},
	},
	"ps1": {
		Extensions:  []string{".iso", ".bin", ".cue"},
		DisplayName: "PlayStation",
		ArtTypes:    []string{"boxart", "screenshot"},
	},
}

// Name returns the source identifier
func (s *Source) Name() string {
	return "emulated"
}

// Init initializes the emulated source
func (s *Source) Init(config map[string]any) error {
	// Set default base path
	s.basePath = filepath.Join(os.Getenv("HOME"), ".local", "share", "gentro", "roms")

	// Override from config
	if config != nil {
		if basePath, ok := config["basePath"].(string); ok && basePath != "" {
			s.basePath = basePath
		}
	}

	// Ensure base path exists
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create ROM base path: %w", err)
	}

	// Set up art cache
	s.artCache = filepath.Join(os.Getenv("HOME"), ".local", "share", "gentro", "cache", "file", "art")
	if err := os.MkdirAll(s.artCache, 0755); err != nil {
		return fmt.Errorf("failed to create art cache path: %w", err)
	}

	// Use default platform configs
	s.platforms = defaultPlatformConfigs

	return nil
}

// GetInstances returns all discovered ROM instances
func (s *Source) GetInstances(ctx context.Context) ([]models.GameInstance, error) {
	var instances []models.GameInstance

	// Walk each platform directory
	for platform := range s.platforms {
		platformPath := filepath.Join(s.basePath, platform)

		// Skip if directory doesn't exist
		if _, err := os.Stat(platformPath); os.IsNotExist(err) {
			continue
		}

		// Walk the platform directory recursively
		err := filepath.Walk(platformPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Check if this is a ROM file
			if !s.isROMFile(path, platform) {
				return nil
			}

			// Create instance
			instance, err := s.createInstance(path, info, platform)
			if err != nil {
				return err
			}

			instances = append(instances, instance)
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan platform %s: %w", platform, err)
		}
	}

	return instances, nil
}

// Refresh rescans the ROM directories and refreshes emulator availability cache
func (s *Source) Refresh(ctx context.Context) error {
	s.populateEmulatorAvailabilityCache()
	return nil
}

// GetGameArt returns art data for a game
func (s *Source) GetGameArt(ctx context.Context, instanceID string, artType string) ([]byte, string, error) {
	// Look for cached art file
	artPath := filepath.Join(s.artCache, instanceID, artType+".png")

	// Check if art exists
	data, err := os.ReadFile(artPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("art not found: %s/%s", instanceID, artType)
		}
		return nil, "", fmt.Errorf("failed to read art: %w", err)
	}

	return data, "image/png", nil
}

// isROMFile checks if a file is a ROM for the given platform
func (s *Source) isROMFile(path string, platform string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	config, ok := s.platforms[platform]
	if !ok {
		return false
	}

	for _, validExt := range config.Extensions {
		if ext == strings.ToLower(validExt) {
			return true
		}
	}
	return false
}

// createInstance creates a GameInstance from a ROM file
func (s *Source) createInstance(path string, info os.FileInfo, platform string) (models.GameInstance, error) {
	// Calculate file hash (first 1MB)
	hash, err := hashFirstMB(path)
	if err != nil {
		return models.GameInstance{}, fmt.Errorf("failed to hash file: %w", err)
	}

	// Generate instance ID from file hash
	instanceID := generateInstanceID(hash)

	// Parse game name from filename
	gameName := parseGameName(info.Name())

	// Generate game ID from name and platform
	gameID := generateGameID(gameName, platform)

	// Check emulator availability from cache or compute on-demand
	hasEmulator := s.getEmulatorAvailabilityForPlatform(platform)

	return models.GameInstance{
		ID:          instanceID,
		GameID:      gameID,
		Source:      "emulated",
		Platform:    platform,
		SourceID:    hash,
		Path:        path,
		Filename:    info.Name(),
		FileSize:    info.Size(),
		FileHash:    hash,
		Installed:   true,
		InstallPath: path,
		CustomMetadata: map[string]any{
			"emulator.available": hasEmulator,
		},
		SourceData: map[string]any{
			"displayName": gameName,
		},
	}, nil
}

// getEmulatorAvailabilityForPlatform returns emulator availability from cache or checks on-demand
func (s *Source) getEmulatorAvailabilityForPlatform(platform string) bool {
	// Return cached value if available
	if s.emulatorAvailabilityCache != nil {
		if available, ok := s.emulatorAvailabilityCache[platform]; ok {
			return available
		}
	}

	// Check on-demand if not in cache
	if s.emuService != nil {
		pairs, err := s.emuService.GetAvailableEmulatorsForPlatform(platform)
		if err == nil && len(pairs) > 0 {
			return true
		}
	}

	return false
}

// hashFirstMB calculates SHA256 hash of the first 1MB of a file
func hashFirstMB(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	// Read first 1MB
	buf := make([]byte, 1024*1024)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	hash.Write(buf[:n])
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// generateInstanceID creates a UUID from file hash
func generateInstanceID(fileHash string) string {
	// Use the file hash directly as ID
	return fmt.Sprintf("file_%s", fileHash[:16])
}

// generateGameID creates a UUID from game name and platform
func generateGameID(name string, platform string) string {
	// Simple hash for now - could use proper UUID generation
	return fmt.Sprintf("game_%s_%s", platform, sanitizeString(name))
}

// parseGameName extracts the game name from filename
func parseGameName(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove common suffixes like region codes, version numbers
	// This is a simple implementation - could be more sophisticated
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	return strings.TrimSpace(name)
}

// sanitizeString makes a string safe for use in IDs
func sanitizeString(s string) string {
	// Replace spaces and special chars
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

// AddManualROM adds a ROM file manually outside the standard structure
func (s *Source) AddManualROM(path string, platform string) (*models.GameInstance, error) {
	// Validate platform
	if _, ok := s.platforms[platform]; !ok {
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a valid ROM
	if !s.isROMFile(path, platform) {
		return nil, fmt.Errorf("file is not a valid ROM for platform %s", platform)
	}

	instance, err := s.createInstance(path, info, platform)
	if err != nil {
		return nil, err
	}

	return &instance, nil
}

// SetEmulatorService injects the emulator service and populates availability cache
func (s *Source) SetEmulatorService(svc *emulator.Service) {
	s.emuService = svc
	s.populateEmulatorAvailabilityCache()
}

// populateEmulatorAvailabilityCache pre-computes emulator availability for all platforms
func (s *Source) populateEmulatorAvailabilityCache() {
	if s.emuService == nil {
		return
	}

	if s.emulatorAvailabilityCache == nil {
		s.emulatorAvailabilityCache = make(map[string]bool)
	}

	for platform := range s.platforms {
		// Check if there's a default emulator available for this platform
		_, _, err := s.emuService.GetDefaultEmulatorForPlatform(platform, true)
		if err == nil {
			s.emulatorAvailabilityCache[platform] = true
			continue
		}

		// Check for any available emulator as fallback
		pairs, err := s.emuService.GetAvailableEmulatorsForPlatform(platform)
		if err == nil && len(pairs) > 0 {
			s.emulatorAvailabilityCache[platform] = true
			continue
		}

		// No emulator available
		s.emulatorAvailabilityCache[platform] = false

		if s.logger != nil {
			s.logger.Warn("no emulator available for platform",
				"platform", platform,
			)
		}
	}
}

// SetLogger injects the logger
func (s *Source) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

// Launch initiates the game using the configured emulator
func (s *Source) Launch(ctx context.Context, instance models.GameInstance) (*exec.Cmd, error) {
	if s.emuService == nil {
		return nil, fmt.Errorf("emulator service not configured")
	}

	// Resolve emulator (platform default or instance override)
	emu, core, err := s.emuService.ResolveEmulator(instance)
	if err != nil {
		return nil, fmt.Errorf("no emulator available for %s: %w", instance.Platform, err)
	}

	if emu == nil {
		return nil, fmt.Errorf("no emulator configured for platform %s", instance.Platform)
	}

	if !emu.IsAvailable {
		return nil, fmt.Errorf("emulator %s is not installed", emu.DisplayName)
	}

	// Log resolved emulator
	if s.logger != nil {
		coreName := ""
		if core != nil {
			coreName = core.DisplayName
		}
		s.logger.Info("resolved emulator",
			"instanceId", instance.ID,
			"emulator", emu.DisplayName,
			"emulatorId", emu.ID,
			"emulatorType", emu.Type,
			"core", coreName,
			"platform", instance.Platform,
		)
	}

	// Get instance-specific settings
	settings, _ := s.emuService.GetInstanceEmulatorSettings(instance.ID)
	customArgs := ""
	if settings != nil {
		customArgs = settings.CustomArgs
	}

	// Build command
	cmd, err := s.emuService.BuildCommand(emu, core, instance.Path, customArgs)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to build emulator command",
				"instanceId", instance.ID,
				"emulator", emu.DisplayName,
				"error", err,
			)
		}
		return nil, fmt.Errorf("failed to build emulator command: %w", err)
	}

	// Get absolute path for ROM (sanitized for logging)
	absRomPath, _ := filepath.Abs(instance.Path)
	if absRomPath == "" {
		absRomPath = instance.Path
	}

	// Log the command being executed (with path sanitized)
	cmdStr := strings.Join(cmd, " ")
	if s.logger != nil {
		// Sanitize home directory from log for privacy
		home := os.Getenv("HOME")
		sanitizedCmd := cmdStr
		if home != "" {
			sanitizedCmd = strings.ReplaceAll(cmdStr, home, "~")
		}
		s.logger.Info("launching game",
			"instanceId", instance.ID,
			"emulator", emu.DisplayName,
			"romPath", absRomPath,
			"command", sanitizedCmd,
		)
	}

	// Execute
	execCmd := exec.Command(cmd[0], cmd[1:]...)

	// Capture stderr for error reporting
	var stderrBuf strings.Builder
	execCmd.Stderr = &stderrBuf

	err = execCmd.Start()
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to start emulator",
				"instanceId", instance.ID,
				"emulator", emu.DisplayName,
				"error", err,
				"command", cmdStr,
			)
		}
		return nil, fmt.Errorf("failed to start emulator: %w", err)
	}

	// Wait a moment and check if process is still running
	time.Sleep(500 * time.Millisecond)
	if execCmd.Process == nil {
		return nil, fmt.Errorf("emulator process failed to start")
	}

	// Check if process has already exited
	if err := execCmd.Process.Signal(syscall.Signal(0)); err != nil {
		stderr := stderrBuf.String()
		if s.logger != nil {
			s.logger.Error("emulator process exited immediately",
				"instanceId", instance.ID,
				"emulator", emu.DisplayName,
				"error", stderr,
			)
		}
		return nil, fmt.Errorf("emulator failed to start: %s", stderr)
	}

	if s.logger != nil {
		s.logger.Info("emulator started successfully",
			"instanceId", instance.ID,
			"emulator", emu.DisplayName,
			"pid", execCmd.Process.Pid,
		)
	}

	return execCmd, nil
}

// MonitorProcess watches the emulator process and emits status events
// For emulated games, we use direct Wait() since we control the process
func (s *Source) MonitorProcess(ctx context.Context, instance models.GameInstance, cmd *exec.Cmd) {
	// Spawn goroutine that blocks on Wait()
	go func() {
		s.logger.Info("starting process monitor",
			"instanceId", instance.ID,
			"pid", cmd.Process.Pid,
		)

		// Emit running immediately - we know process started successfully
		s.emitRunning(instance)

		// Wait for process to exit (blocking)
		err := cmd.Wait()

		if err != nil {
			s.logger.Error("emulator process exited with error",
				"instanceId", instance.ID,
				"error", err,
			)
		} else {
			s.logger.Info("emulator process exited normally",
				"instanceId", instance.ID,
			)
		}

		// Emit stopped immediately when Wait() returns
		s.emitStopped(instance)
	}()
}

// emitRunning emits a running status update
func (s *Source) emitRunning(instance models.GameInstance) {
	app := application.Get()
	if app != nil {
		update := models.LaunchStatusUpdate{
			InstanceID: instance.ID,
			GameID:     instance.GameID,
			Status:     models.LaunchStatusRunning,
		}
		app.Event.Emit("launchStatusUpdate", update)
	}

	if s.logger != nil {
		s.logger.Info("game running",
			"instanceId", instance.ID,
			"gameId", instance.GameID,
		)
	}
}

// emitStopped emits a stopped status update
func (s *Source) emitStopped(instance models.GameInstance) {
	app := application.Get()
	if app != nil {
		update := models.LaunchStatusUpdate{
			InstanceID: instance.ID,
			GameID:     instance.GameID,
			Status:     models.LaunchStatusStopped,
		}
		app.Event.Emit("launchStatusUpdate", update)
	}

	if s.logger != nil {
		s.logger.Info("game stopped",
			"instanceId", instance.ID,
			"gameId", instance.GameID,
		)
	}
}

// FilterInstances applies emulated source-specific filters
// Currently no specific filters for emulated games
func (s *Source) FilterInstances(instances []models.GameInstance, filter models.GameFilter) []models.GameInstance {
	// Emulated source has no specific filters yet
	return instances
}
