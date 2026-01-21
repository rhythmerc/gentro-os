# Gentro OS

Console-first, immutable live-USB Linux distro focused on gaming. Gentro OS combines a custom
launcher UI, a plugin-driven game library, and an in-game overlay to manage emulator settings and
launch Steam titles without relying on Steam Big Picture mode.

## Goals
- Live USB boot with persistent data
- Immutable base OS (Fedora Kinoite + OSTree)
- Gamescope + Wayland session
- Tauri-based UI shell and overlay
- Emulator support via custom Flatpaks

## Repo layout
- launcher-ui: Tauri UI shell and overlay
- launcher-core: game library, plugin system, and configuration logic
- system-services: IPC bridge, launcher services, and system daemons
- plan: project planning and architecture docs
- scripts: local dev helpers

## Development
Run the launcher-core daemon in Docker:

```bash
./scripts/dev-core
```

For more details, see `plan/dev-environment.md`.

## Status
Planning and early scaffolding.
