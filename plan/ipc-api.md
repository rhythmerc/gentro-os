# IPC API (JSON-RPC)

## Transport
- Unix domain socket (default)
- HTTP JSON-RPC at `/rpc` for macOS dev
- JSON-RPC 2.0
- UTF-8
- Single request/response per connection (v1), optional persistent connections later

## Conventions
- All methods prefixed by service namespace
- Standard error object for unsupported operations
- All settings are declared with a capability entry

## Service: overlay

### overlay.show
Show overlay and steal input focus.

Request
```json
{"jsonrpc":"2.0","id":1,"method":"overlay.show","params":{}}
```

Response
```json
{"jsonrpc":"2.0","id":1,"result":{"visible":true}}
```

### overlay.hide
Hide overlay and return input to game.

### overlay.toggle
Toggle overlay visibility.

### overlay.status
Query overlay visibility and focus state.

Result
```json
{"visible":true,"focus":"ui"}
```

## Service: emulator

### emulator.capabilities
Return supported settings and whether they are live-changeable.

Result
```json
{
  "emulator":"dolphin",
  "settings":[
    {"key":"gfx.backend","type":"string","live":false},
    {"key":"gfx.internal_resolution","type":"int","live":true},
    {"key":"input.profile.wiimote.1","type":"string","live":true}
  ]
}
```

### emulator.get
Get current value for a setting.

Request
```json
{"jsonrpc":"2.0","id":2,"method":"emulator.get","params":{"key":"gfx.internal_resolution"}}
```

Response
```json
{"jsonrpc":"2.0","id":2,"result":{"key":"gfx.internal_resolution","value":3}}
```

### emulator.set
Set a setting. Returns whether it was applied live or deferred.

Params
- key: string
- value: any
- scope: "emulator" | "game" (defaults to "game")

Request
```json
{"jsonrpc":"2.0","id":3,"method":"emulator.set","params":{"key":"gfx.internal_resolution","value":4,"scope":"game"}}
```

Response
```json
{"jsonrpc":"2.0","id":3,"result":{"key":"gfx.internal_resolution","applied":"live","scope":"game"}}
```

### emulator.apply
Apply pending deferred settings (if supported).

Params
- scope: "emulator" | "game" | "all" (defaults to "game")

### emulator.reset
Reset settings to defaults (scope may be per-game or global).

## Service: library

### library.list
Return list of games with metadata and install state.

### library.install
Delegate installation to plugin and return a job id.

### library.uninstall
Delegate uninstall to plugin and return a job id.

### library.launch
Launch game by id.

## Service: jobs

### jobs.status
Query async job progress (install/update/download).

Result
```json
{"job_id":"abc123","state":"running","progress":0.42}
```

## Error format
```json
{
  "jsonrpc":"2.0",
  "id":3,
  "error":{"code":-32601,"message":"Method not found"}
}
```
