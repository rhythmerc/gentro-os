# Dolphin Flatpak Pipeline Checklist

## Goals
- Produce a custom Dolphin Flatpak with patches for in-process IPC
- Integrate /data access for ROMs, saves, and mods
- Enable fast rebuild and local testing

## Build strategy
- Base off org.DolphinEmu.dolphin-emu if possible
- Maintain a forked Flatpak manifest in your repo
- Add a custom patch set for IPC server and config hooks

## Checklist

### Repository setup
- Create `flatpak-manifests/dolphin/`
- Add manifest (`dolphin.yml` or `dolphin.json`)
- Define runtime and SDK (likely org.freedesktop.Platform)

### Build tooling
- Install `flatpak-builder` on build host
- Configure local Flatpak repo
- Add a helper script to build, install, and run Dolphin

### Patching Dolphin
- Add IPC server (JSON-RPC over Unix socket)
- Expose Config::Set* endpoints for live changes
- Provide capability listing endpoint
- Ensure socket path is configurable for sandboxing

### Filesystem access
- Provide read/write access to /data/roms, /data/saves, /data/mods
- Validate save file persistence across runs
- Avoid writing to /home unless explicitly needed

### Permissions
- GPU access (DRI)
- Input devices access
- Audio (PipeWire or PulseAudio)
- Network (required for updates/metadata)

### Local testing
- Launch Dolphin from the UI or via CLI
- Connect to IPC socket and set a live config value
- Validate overlay can steal input and return it

### Distribution
- Publish to a custom Flatpak repo
- Add repo to the base OS image
- Test update and rollback behavior

## Open questions
- Use a custom Flatpak runtime or upstream runtime?
- Best way to grant exFAT /data access under sandbox constraints
- Maintain a single Flatpak for all systems or per-emulator packages
