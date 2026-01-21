# Dev Environment

## Docker (recommended)
Use Docker for `launcher-core` while keeping `launcher-ui` on the host.

### Start the core daemon
```bash
docker compose up --build launcher-core
```

### Socket, data, and logs
The Docker configuration maps all state into the repo under `.gentro/`.

Paths inside the container:
- Socket: `/workspace/.gentro/run/launcher.sock`
- Data: `/workspace/.gentro/data`
- Logs: `/workspace/.gentro/logs/gentro-core.log`

On the host, those appear under:
- `.gentro/run/launcher.sock`
- `.gentro/data`
- `.gentro/logs/gentro-core.log`

### Test with gentroctl
```bash
cargo run --bin gentroctl -- --socket .gentro/run/launcher.sock core.status
```

On macOS, Docker Desktop runs inside a VM, so host Unix sockets cannot connect to container
listeners. Use HTTP JSON-RPC over port 8123:

```bash
curl -s http://localhost:8123/rpc \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"core.status","params":{}}'
```

The launcher UI can use the same endpoint via:

```bash
VITE_GENTRO_RPC_URL=http://localhost:8123/rpc npm run dev
```

Allowed dev origins for HTTP JSON-RPC:
- http://localhost:5173
- http://localhost:1420
- tauri://localhost

You can also run gentroctl inside the container:

```bash
docker compose exec -T launcher-core cargo run --bin gentroctl -- \
  --socket /workspace/.gentro/run/launcher.sock core.status
```

## Distrobox (optional)
Use Distrobox when you need Fedora tooling (rpm-ostree, Flatpak behavior, Gamescope testing)
without a full VM.

### Create a Fedora environment
```bash
distrobox create -n gentro-fedora -i fedora:40
```

### Enter the box
```bash
distrobox enter gentro-fedora
```

### Why this matters later
- Validate OSTree workflows and Kinoite behaviors.
- Test system services and permissions closer to production.
- Experiment with Flatpak sandboxes and host integrations.

## Notes
- Tauri UI is best run on the host during dev.
- Use the dockerized daemon for IPC and storage consistency.
