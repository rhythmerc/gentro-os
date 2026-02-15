package emulator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rhythmerc/gentro-ui/services/games/database"
	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// Service manages emulator discovery and configuration
type Service struct {
	db     *database.DB
	logger Logger
}

// Logger interface for logging
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

// NewService creates a new emulator service
func NewService(db *database.DB, logger Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

// Initialize seeds default emulators and mappings
func (s *Service) Initialize() error {
	s.logger.Info("Initializing emulator service")

	// Seed default emulators
	for _, emu := range DefaultEmulators() {
		if err := s.db.UpsertEmulator(emu); err != nil {
			return fmt.Errorf("failed to seed emulator %s: %w", emu.ID, err)
		}
		s.logger.Info("Seeded emulator", "id", emu.ID)
	}

	// Seed default cores
	for _, core := range DefaultCores() {
		if err := s.db.UpsertEmulatorCore(core); err != nil {
			return fmt.Errorf("failed to seed core %s: %w", core.ID, err)
		}
		s.logger.Info("Seeded core", "id", core.ID)
	}

	// Seed platform mappings
	for _, mapping := range DefaultPlatformMappings() {
		if err := s.db.UpsertPlatformEmulator(mapping); err != nil {
			return fmt.Errorf("failed to seed platform mapping %s: %w", mapping.ID, err)
		}
		s.logger.Info("Seeded platform mapping", "id", mapping.ID)
	}

	return nil
}

// DiscoverAvailable scans for installed emulators and cores
func (s *Service) DiscoverAvailable() error {
	s.logger.Info("Discovering available emulators")

	// Get all emulators
	emulators, err := s.db.GetEmulators()
	if err != nil {
		return fmt.Errorf("failed to get emulators: %w", err)
	}

	// Check each emulator
	for _, emu := range emulators {
		available := false
		if emu.Type == models.EmulatorTypeFlatpak {
			available = s.checkFlatpakInstalled(emu.FlatpakID)
		} else if emu.Type == models.EmulatorTypeNative {
			available = s.checkNativeInstalled(emu.ExecutablePath)
		}

		if available != emu.IsAvailable {
			s.db.UpdateEmulatorAvailability(emu.ID, available)
			s.logger.Info("Updated emulator availability", "id", emu.ID, "available", available)
		}
	}

	// If RetroArch is available, discover its cores
	retroarch := s.getEmulatorByID(emulators, "retroarch")
	if retroarch != nil && retroarch.IsAvailable {
		s.discoverRetroArchCores()
	}

	return nil
}

func (s *Service) checkFlatpakInstalled(flatpakID string) bool {
	if flatpakID == "" {
		return false
	}
	cmd := exec.Command("flatpak", "info", flatpakID)
	err := cmd.Run()
	return err == nil
}

func (s *Service) checkNativeInstalled(executablePath string) bool {
	if executablePath == "" {
		return false
	}
	_, err := exec.LookPath(executablePath)
	return err == nil
}

func (s *Service) getEmulatorByID(emulators []models.Emulator, id string) *models.Emulator {
	for _, emu := range emulators {
		if emu.ID == id {
			return &emu
		}
	}
	return nil
}

func (s *Service) discoverRetroArchCores() {
	s.logger.Info("Discovering RetroArch cores")

	// Check for cores in the Flatpak directory
	coresPath := filepath.Join(
		os.Getenv("HOME"),
		".var", "app", "org.libretro.RetroArch",
		"config", "retroarch", "cores",
	)

	cores, err := s.db.GetEmulatorCores("retroarch")
	if err != nil {
		s.logger.Error("Failed to get RetroArch cores", "error", err)
		return
	}

	for _, core := range cores {
		corePath := filepath.Join(coresPath, core.CoreID+".so")
		available := false
		if _, err := os.Stat(corePath); err == nil {
			available = true
		}

		if available != core.IsAvailable {
			s.db.UpdateEmulatorCoreAvailability(core.ID, available)
			s.logger.Info("Updated core availability", "id", core.ID, "available", available)
		}
	}
}

// ResolveEmulator finds the appropriate emulator for a game instance
func (s *Service) ResolveEmulator(instance models.GameInstance) (*models.Emulator, *models.EmulatorCore, error) {
	s.logger.Info("resolving emulator",
		"instanceId", instance.ID,
		"platform", instance.Platform,
	)

	// 1. Check instance override
	settings, err := s.db.GetInstanceEmulatorSettings(instance.ID)
	if err == nil && settings != nil {
		s.logger.Info("using instance-specific emulator settings",
			"instanceId", instance.ID,
			"emulatorId", settings.EmulatorID,
			"coreId", settings.CoreID,
		)
		return s.getEmulatorAndCore(settings.EmulatorID, settings.CoreID)
	}

	// 2. Check platform default
	emu, core, err := s.db.GetDefaultEmulatorForPlatform(instance.Platform)
	if err != nil {
		s.logger.Error("no default emulator found",
			"instanceId", instance.ID,
			"platform", instance.Platform,
			"error", err,
		)
		return nil, nil, err
	}

	if emu != nil {
		coreName := ""
		if core != nil {
			coreName = core.DisplayName
		}
		s.logger.Info("using platform default emulator",
			"instanceId", instance.ID,
			"platform", instance.Platform,
			"emulator", emu.DisplayName,
			"core", coreName,
		)
	}

	return emu, core, nil
}

func (s *Service) getEmulatorAndCore(emulatorID, coreID string) (*models.Emulator, *models.EmulatorCore, error) {
	emulator, err := s.db.GetEmulator(emulatorID)
	if err != nil {
		return nil, nil, fmt.Errorf("emulator not found: %s", emulatorID)
	}

	var core *models.EmulatorCore
	if coreID != "" {
		c, err := s.db.GetCore(emulatorID, coreID)
		if err == nil {
			core = c
		}
	}

	return emulator, core, nil
}

// BuildCommand constructs the launch command for an emulator
func (s *Service) BuildCommand(emulator *models.Emulator, core *models.EmulatorCore, romPath string, customArgs string) ([]string, error) {
	if emulator == nil {
		return nil, fmt.Errorf("emulator is nil")
	}

	if !emulator.IsAvailable {
		return nil, fmt.Errorf("emulator %s is not available", emulator.ID)
	}

	// Get the core library path if using RetroArch
	var coreLibPath string
	if core != nil && emulator.ID == "retroarch" {
		coreLibPath = filepath.Join(
			os.Getenv("HOME"),
			".var", "app", "org.libretro.RetroArch",
			"config", "retroarch", "cores",
			core.CoreID+".so",
		)
		s.logger.Info("using RetroArch core",
			"coreId", core.CoreID,
			"coreLibPath", coreLibPath,
		)
	}

	// Combine default args with custom args
	args := emulator.DefaultArgs
	if customArgs != "" {
		if args != "" {
			args = args + " " + customArgs
		} else {
			args = customArgs
		}
	}

	s.logger.Info("building command",
		"emulator", emulator.ID,
		"emulatorType", emulator.Type,
		"template", emulator.CommandTemplate,
		"coreLibPath", coreLibPath,
		"romPath", romPath,
		"args", args,
	)

	// Build command based on emulator type
	if emulator.Type == models.EmulatorTypeFlatpak {
		return s.buildFlatpakCommand(emulator, coreLibPath, romPath, args), nil
	}

	return s.buildNativeCommand(emulator, romPath, args), nil
}

func (s *Service) buildFlatpakCommand(emulator *models.Emulator, coreLibPath, romPath, args string) []string {
	// Quote paths that contain spaces
	quotedRomPath := quotePathIfNeeded(romPath)
	quotedCorePath := quotePathIfNeeded(coreLibPath)

	// Template substitution
	cmd := emulator.CommandTemplate
	cmd = strings.ReplaceAll(cmd, "{flatpak_id}", emulator.FlatpakID)
	cmd = strings.ReplaceAll(cmd, "{core_lib_path}", quotedCorePath)
	cmd = strings.ReplaceAll(cmd, "{args}", args)
	cmd = strings.ReplaceAll(cmd, "{rom}", quotedRomPath)

	// Parse into slice, but handle quoted strings properly
	return parseCommandWithQuotes(cmd)
}

func (s *Service) buildNativeCommand(emulator *models.Emulator, romPath, args string) []string {
	// Quote paths that contain spaces
	quotedRomPath := quotePathIfNeeded(romPath)

	cmd := emulator.CommandTemplate
	cmd = strings.ReplaceAll(cmd, "{executable}", emulator.ExecutablePath)
	cmd = strings.ReplaceAll(cmd, "{args}", args)
	cmd = strings.ReplaceAll(cmd, "{rom}", quotedRomPath)

	// Parse into slice, but handle quoted strings properly
	return parseCommandWithQuotes(cmd)
}

// GetEmulators returns all emulators
func (s *Service) GetEmulators() ([]models.Emulator, error) {
	return s.db.GetEmulators()
}

// GetEmulatorsForPlatform returns emulators available for a platform
func (s *Service) GetEmulatorsForPlatform(platform string) ([]models.Emulator, []models.EmulatorCore, error) {
	return s.db.GetEmulatorsForPlatform(platform)
}

// SetPlatformDefault sets the default emulator for a platform
func (s *Service) SetPlatformDefault(platform, emulatorID, coreID string) error {
	return s.db.SetPlatformDefaultEmulator(platform, emulatorID, coreID)
}

// SetInstanceEmulator sets the emulator for a specific game instance
func (s *Service) SetInstanceEmulator(instanceID, emulatorID, coreID, customArgs string) error {
	return s.db.SetInstanceEmulatorSettings(instanceID, emulatorID, coreID, customArgs)
}

// quotePathIfNeeded wraps a path in quotes if it contains spaces
func quotePathIfNeeded(path string) string {
	if strings.Contains(path, " ") {
		return fmt.Sprintf("%q", path)
	}
	return path
}

// parseCommandWithQuotes parses a command string, respecting quoted arguments
func parseCommandWithQuotes(cmd string) []string {
	var args []string
	var currentArg strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range cmd {
		switch {
		case r == '"' || r == '\'':
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				currentArg.WriteRune(r)
			}
		case r == ' ' && !inQuote:
			if currentArg.Len() > 0 {
				args = append(args, currentArg.String())
				currentArg.Reset()
			}
		default:
			currentArg.WriteRune(r)
		}
	}

	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	return args
}

// GetInstanceEmulatorSettings retrieves emulator settings for an instance
func (s *Service) GetInstanceEmulatorSettings(instanceID string) (*models.InstanceEmulatorSettings, error) {
	return s.db.GetInstanceEmulatorSettings(instanceID)
}
