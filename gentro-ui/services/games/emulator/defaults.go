package emulator

import "github.com/rhythmerc/gentro-ui/services/games/models"

// DefaultEmulators returns pre-configured emulator definitions
func DefaultEmulators() []models.Emulator {
	return []models.Emulator{
		{
			ID:              "retroarch",
			Name:            "retroarch",
			DisplayName:     "RetroArch",
			Type:            models.EmulatorTypeFlatpak,
			FlatpakID:       "org.libretro.RetroArch",
			CommandTemplate: "flatpak run {flatpak_id} -L {core_lib_path} {args} {rom}",
			DefaultArgs:     "--fullscreen",
		},
		{
			ID:              "nestopia",
			Name:            "nestopia",
			DisplayName:     "Nestopia UE",
			Type:            models.EmulatorTypeFlatpak,
			FlatpakID:       "ca._0ldsk00l.Nestopia",
			CommandTemplate: "flatpak run {flatpak_id} {args} {rom}",
			DefaultArgs:     "--fullscreen",
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

// DefaultPlatformMappings returns platform -> emulator/core defaults
func DefaultPlatformMappings() []models.PlatformEmulator {
	return []models.PlatformEmulator{
		{
			ID:         "nes_retroarch_mesen",
			Platform:   "nes",
			EmulatorID: "retroarch",
			CoreID:     "mesen_libretro",
			IsDefault:  true,
			Priority:   0,
		},
		{
			ID:         "nes_retroarch_nestopia",
			Platform:   "nes",
			EmulatorID: "retroarch",
			CoreID:     "nestopia_libretro",
			IsDefault:  false,
			Priority:   1,
		},
		{
			ID:         "nes_nestopia_standalone",
			Platform:   "nes",
			EmulatorID: "nestopia",
			IsDefault:  false,
			Priority:   2,
		},
		{
			ID:         "snes_retroarch_snes9x",
			Platform:   "snes",
			EmulatorID: "retroarch",
			CoreID:     "snes9x_libretro",
			IsDefault:  true,
			Priority:   0,
		},
	}
}
