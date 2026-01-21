# Gentro OS Planning Overview

## Vision
Build a console-first, live-USB Linux distro optimized for gaming with a unified launcher UI that
exposes Steam and emulated libraries, supports an in-game overlay, and provides stable, immutable
system updates.

## Guiding principles
- Console-centric UX with a controller-first interaction model.
- Immutable base OS with reliable rollback and predictable behavior.
- Strong overlay model for in-game configuration changes.
- Emulators treated as first-class systems with unified config and metadata.
- Cross-platform media accessibility for ROMs, saves, and mods.
- Per-game overrides owned by the system, not emulator INI overrides.
- macOS dev uses HTTP JSON-RPC due to Docker Desktop socket limits.

## Primary requirements
- Live USB is the primary boot mode; installation to disk is secondary.
- Persistence is a top priority; fast boot and updates follow.
- Gamescope + Wayland stack; overlay must be able to steal input fully.
- Steam is a primary experience, but not the default UI.
- Dolphin is the first emulator to support live config changes.

## High-level scope
- Base OS: Fedora Kinoite (immutable, OSTree).
- UI shell: Tauri (Rust + web UI).
- IPC: JSON-RPC over Unix sockets.
- App packaging: Custom Flatpaks for emulators.
- Storage: exFAT data partition for external access.

## Non-goals (initial)
- ARM support in v1 (planned for later).
- Full installer and disk partitioning UI.
- Wide emulator catalog before Dolphin integration is stable.
- Heavy reliance on Steam Big Picture mode.

## Open questions
- How much of Dolphin live-config requires upstream patches vs extension hooks?
- How to handle DRM-protected content and codecs in a live environment?
- Best overlay compositing strategy with Gamescope when launching Steam titles.
