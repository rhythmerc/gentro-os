# launcher-core

Rust daemon for Gentro OS. Provides JSON-RPC services over a Unix socket, manages the game library
database, and exposes emulator configuration controls for the UI layer.

## Binaries
- `launcher-core` (daemon)
- `gentroctl` (CLI client)

## Run (dev)
```bash
cargo run
```

## Test JSON-RPC
```bash
cargo run --bin gentroctl -- core.status
cargo run --bin gentroctl -- -p '{"key":"gfx.internal_resolution","value":3}' emulator.set
```

## Environment overrides
- `GENTRO_SOCKET_PATH` (default: `/run/gentro/launcher.sock`)
- `GENTRO_TCP_ADDR` (optional, enables HTTP JSON-RPC)
- `GENTRO_DATA_DIR` (default: `/data/gentro`)
- `GENTRO_LOG_PATH` (default: `/data/logs/gentro-core.log`)
