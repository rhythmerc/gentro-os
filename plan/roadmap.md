# Roadmap

## Phase 0: Decisions and specs
- Confirm base OS choice: Fedora Kinoite
- Define IPC schema (JSON-RPC methods and payloads)
- Draft plugin interfaces and core game library schema
- Select minimal Dolphin settings for live update prototype
- Define emulator-specific, additive per-game override model

## Phase 1: Base OS prototype
- Boot Fedora Kinoite into a bare Gamescope session
- Validate GPU, Vulkan, audio, controller, network
- Prove persistent data partition mount
- Basic Tauri shell running fullscreen
- Define macOS dev transport for launcher-core (HTTP JSON-RPC)

## Phase 2: Flatpak pipeline
- Stand up a custom Flatpak repo
- Build and install a custom Dolphin Flatpak
- Verify access to /data for ROMs/saves/mods

## Phase 3: Overlay + input
- Implement overlay toggle and input stealing
- Route controller input to UI when overlay is active
- Validate Steam launch flow without Big Picture

## Phase 4: Dolphin live config
- Add in-process IPC server to Dolphin
- Implement JSON-RPC endpoints for a few settings
- Establish live vs restart-required capability matrix

## Phase 5: Library plugin system
- Local ROM plugin indexing /data/roms
- Core library database with installed/uninstalled state
- Placeholder Steam plugin scaffolding

## Phase 6: Updates and recovery
- OSTree update channel with rollback
- Flatpak updates for emulators
- Recovery mode and factory reset for /data

## Phase 7: Performance and polish
- Boot time optimization
- UX polish, branding, and audio cues
- Expand emulator coverage and plugin sources
