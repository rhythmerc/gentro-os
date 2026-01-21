# Dolphin Live-Config Capability Matrix (Draft)

## Notes
- Live-changeable settings depend on Dolphin runtime hooks and backend support.
- This is a starting hypothesis based on code structure and typical emulator behavior.
- Each entry should be validated during integration.

## Legend
- live: Change applies immediately
- restart: Change requires emulator restart
- game: Change applies on next game start

## Graphics
- gfx.backend (OpenGL/Vulkan): restart
- gfx.internal_resolution: live
- gfx.vsync: live
- gfx.antialiasing: live
- gfx.aspect_ratio: live
- gfx.shader_compilation_mode: restart
- gfx.shader_precompile: restart
- gfx.texture_filtering: live
- gfx.widescreen_hack: live

## Performance
- core.cpu_engine (JIT/Interpreter): restart
- core.dual_core: restart
- core.sync_on_skip_idle: restart
- core.overclock: live
- core.emulation_speed: live

## Audio
- audio.backend: restart
- audio.volume: live
- audio.stretching: live
- audio.latency: restart

## Input
- input.profile.gc.1: live
- input.profile.wiimote.1: live
- input.rumble: live
- input.deadzone: live

## Wii
- wii.sd_card_inserted: live
- wii.speak_muted: live
- wii.keyboard_connected: live

## Game-specific
- system-managed overrides (cheats/patches) are handled outside Dolphin

## Next validation steps
- Confirm which settings are mapped to Config::Set* live callbacks
- Identify any settings that require per-frame refresh or explicit reload hooks
- Track video backend-specific behavior (Vulkan vs OpenGL)
- Ensure emulator INI overrides are not relied on for per-game settings
