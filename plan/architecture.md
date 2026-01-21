# Architecture

## System layers
- Immutable system (OSTree)
  - Kernel, Mesa/Vulkan, input stack, audio stack, Gamescope
  - Launcher services and core system daemons
- Mutable app layer
  - Custom Flatpaks for emulators and auxiliary tools
  - Flatpak runtime updates independent of base OS

## Boot and session flow
1) systemd
2) autologin or greetd session
3) gamescope-session
4) Gentro UI shell
5) game/emulator launched inside Gamescope

## UI stack
- Tauri app as the primary shell and overlay
- UI always running; foreground toggled via controller shortcut
- Input focus toggles between overlay and game

## Update model
- Base OS: OSTree A/B updates with rollback
- Applications: Flatpak repo updates
- Data: persistent partition preserved across updates

## Storage model
- Partitions
  - system: immutable OSTree
  - data: exFAT for ROMs, saves, mods, metadata
- Standard paths
  - /data/roms/<system>/
  - /data/saves/<system>/
  - /data/mods/<system>/
  - /data/media/
  - /data/config/

## Game library model
- Core entities
  - Game: id, title, platform, source, install_state, launch_target
  - Source: plugin-provided catalog (Local ROM, Steam, others)
  - Install: handles installed/uninstalled states and download progress
- Plugin architecture
  - list_games()
  - install(game)
  - uninstall(game)
  - launch(game)
  - get_install_state(game)
- Emulator-specific overrides
  - Per-emulator defaults stored by the system
  - Per-game overrides stored as additive deltas
  - Multiple emulators per system supported via selected emulator id

## Overlay and emulator control
- Overlay steals input fully when active
- JSON-RPC over Unix socket for UI <-> system services
- Dolphin integration via in-process IPC server
  - Config changes applied via Config::Set* to trigger live callbacks
  - Capability matrix indicating live-change vs restart-required
- Per-game overrides applied by system adapters, not emulator Game INI

## Steam integration
- Steam client is a primary source but not UI owner
- UI reads Steam manifests or API via plugin
- Launch via steam://run/<id> or dedicated launch service
