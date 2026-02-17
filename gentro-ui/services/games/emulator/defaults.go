package emulator

import "github.com/rhythmerc/gentro-ui/services/games/models"

// DefaultEmulatorConfig specifies which emulator/core is the default for a platform
type DefaultEmulatorConfig struct {
	EmulatorID string
	CoreID     string // empty for standalone emulators
}

// DefaultEmulatorsByPlatform maps platforms to their default emulator configuration
var DefaultEmulatorsByPlatform = map[string]DefaultEmulatorConfig{
	"nes":  {EmulatorID: "retroarch", CoreID: "mesen_libretro"},
	"snes": {EmulatorID: "retroarch", CoreID: "snes9x_libretro"},
	"wii":  {EmulatorID: "dolphin"},
}

// DefaultEmulators returns pre-configured emulator definitions
func DefaultEmulators() []models.Emulator {
	return []models.Emulator{
		{
			ID:                 "retroarch",
			Name:               "retroarch",
			DisplayName:        "RetroArch",
			Type:               models.EmulatorTypeFlatpak,
			FlatpakID:          "org.libretro.RetroArch",
			CommandTemplate:    "flatpak run {flatpak_id} -L {core_lib_path} {args} {rom}",
			DefaultArgs:        "--fullscreen",
			SupportedPlatforms: []string{}, // Cores define platforms, not the emulator itself
		},
		{
			ID:                 "nestopia",
			Name:               "nestopia",
			DisplayName:        "Nestopia UE",
			Type:               models.EmulatorTypeFlatpak,
			FlatpakID:          "ca._0ldsk00l.Nestopia",
			CommandTemplate:    "flatpak run {flatpak_id} {args} {rom}",
			DefaultArgs:        "--fullscreen",
			SupportedPlatforms: []string{"nes"},
		},
		{
			ID:                 "dolphin",
			Name:               "dolphin",
			DisplayName:        "Dolphin",
			Type:               models.EmulatorTypeFlatpak,
			FlatpakID:          "org.DolphinEmu.dolphin-emu",
			CommandTemplate:    "flatpak run {flatpak_id} {args} {rom}",
			DefaultArgs:        "-b -e",
			SupportedPlatforms: []string{"wii", "gamecube"},
		},
	}
}

// DefaultCores returns pre-configured RetroArch cores (Option B)
func DefaultCores() []models.EmulatorCore {
	return []models.EmulatorCore{
		{
			ID:                 "retroarch_mesen",
			EmulatorID:         "retroarch",
			CoreID:             "mesen_libretro",
			DisplayName:        "Mesen",
			SupportedPlatforms: []string{"nes"},
		},
		{
			ID:                 "retroarch_nestopia",
			EmulatorID:         "retroarch",
			CoreID:             "nestopia_libretro",
			DisplayName:        "Nestopia",
			SupportedPlatforms: []string{"nes"},
		},
		{
			ID:                 "retroarch_snes9x",
			EmulatorID:         "retroarch",
			CoreID:             "snes9x_libretro",
			DisplayName:        "Snes9x",
			SupportedPlatforms: []string{"snes"},
		},
		{
			ID:                 "retroarch_bsnes",
			EmulatorID:         "retroarch",
			CoreID:             "bsnes_libretro",
			DisplayName:        "bsnes",
			SupportedPlatforms: []string{"snes"},
		},
	}
}
