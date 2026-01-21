# Next Steps (Tomorrow)

## Recommended plan
- Confirm the dev workflow commands for macOS (Docker core + HTTP JSON-RPC)
- Clean up any remaining launcher-core warnings and add a simple health check script
- Move the UI RPC fetch into a shared helper module (`launcher-ui/src/lib/rpc.ts`)
- Implement a minimal `library.list` in launcher-core using SQLite
- Add a simple “0 games” list view in the UI wired to `library.list`

## Optional stretch
- Add a `scripts/core-status` helper for quick health checks
- Expand launcher-core README to document macOS HTTP RPC usage
