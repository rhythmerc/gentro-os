# AGENTS.md - Coding Guidelines for gentro-ui

## Project Overview

A Wails3 application (Go backend + Svelte5 frontend) for managing game libraries. Uses SQLite for data persistence and supports multiple game sources (Steam, local files).

## Build Commands

### Development
```bash
# Run in development mode (hot reload for Go + frontend)
task dev

# Or use wails3 directly
wails3 dev -config ./build/config.yml -port 9245

# Run frontend only (Vite dev server)
cd frontend && npm run dev
```

### Production Build
```bash
# Build for current OS
task build

# Build for specific OS
task linux:build      # Linux
task windows:build    # Windows
task darwin:build     # macOS

# Server mode (HTTP only, no GUI)
task build:server
task run:server
```

### Frontend Only
```bash
cd frontend
npm run build         # Production build
npm run build:dev     # Development build (no minify)
npm run check         # TypeScript/Svelte type check
npm run preview       # Preview production build
```

### Go Commands
```bash
# Standard Go commands
go mod tidy
go build .
go run .

# Generate Wails bindings (auto-run by task)
wails3 generate bindings -clean=true -ts

# Testing
go test ./...
go test -v ./services/...
go test -run TestFunctionName ./path/to/package
```

## Lint/Test Commands

### Go
```bash
# Format code (REQUIRED before commits)
go fmt ./...

# Vet code for issues
go vet ./...

# Run tests
go test ./...
go test -v ./services/games/...

# Run single test
go test -v -run TestFunctionName ./path/to/package
```

### Frontend
```bash
cd frontend

# Type checking (Svelte + TypeScript)
npm run check
npm run check:watch   # Watch mode

# Note: No ESLint/Prettier configured - rely on IDE formatting
```

## Code Style Guidelines

### Go

- **Formatting**: Use `go fmt` - no custom formatting rules
- **Imports**: Group standard library, third-party, then internal packages with blank lines between
- **Error Handling**: Wrap errors with `fmt.Errorf("...: %w", err)` for context
- **Naming**: 
  - PascalCase for exported identifiers
  - camelCase for unexported
  - Acronyms in ALLCAPS (URL, HTTP, ID)
- **Comments**: Document all exported functions, types, and packages
- **Structs**: Use tags for JSON and DB: `json:"fieldName" db:"field_name"`

Example:
```go
package games

import (
	"fmt"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// GamesService manages games from multiple sources
type GamesService struct {
	db     *database.DB
	logger *slog.Logger
}

// NewGamesService creates a new GamesService
func NewGamesService(config GamesServiceConfig) (*GamesService, error) {
	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	// ...
}
```

### TypeScript / Svelte

- **Version**: Svelte 5 with runes (`$state`, `$derived`, etc.)
- **TypeScript**: Strict mode enabled, always define types for exports
- **Imports**: Use `@wailsio/runtime` for Wails bindings, `$lib` for internal
- **Components**: Use Svelte 5 runes, not legacy `$:` reactive statements
- **Naming**: 
  - PascalCase for components
  - camelCase for variables/functions
  - Types/interfaces in PascalCase with descriptive names

Example:
```svelte
<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import type { Game } from "$lib/types";
  
  // Props with types
  let { game }: { game: Game } = $props();
  
  // State with runes
  let loading = $state(false);
  let games = $state<Game[]>([]);
</script>
```

## Project Structure

```
├── main.go                    # Entry point
├── go.mod                     # Go module definition
├── Taskfile.yml               # Main task definitions
├── build/                     # Build configs per platform
│   ├── Taskfile.yml          # Common build tasks
│   ├── config.yml            # Wails3 configuration
│   └── {linux,darwin,windows}/
├── frontend/                 # Svelte frontend
│   ├── src/
│   │   ├── routes/          # SvelteKit routes
│   │   ├── lib/             # Shared components/utilities
│   │   └── app.d.ts         # Type declarations
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.ts
│   └── tsconfig.json
├── services/                 # Go business logic
│   └── games/
│       ├── games.go         # Main service
│       ├── models/          # Data models
│       ├── database/        # SQLite operations
│       ├── metadata/        # Metadata fetching
│       └── sources/         # Game sources (steam, file)
└── bin/                     # Build output
```

## Key Patterns

### Go Services
- Services implement `ServiceStartup(ctx, options)` and `ServiceShutdown(ctx)`
- Use `application.NewService()` to register with Wails
- Log with `slog.Logger` passed via config
- Return wrapped errors: `fmt.Errorf("context: %w", err)`

### Frontend Bindings
- Auto-generated in `frontend/bindings/`
- Import from `@wailsio/runtime` for events
- Use Wails event system for async updates: `Events.On('eventName', handler)`

### Database
- SQLite with `mattn/go-sqlite3`
- Migration logic in `database.go`
- JSON fields stored as text, parsed manually

## Important Notes

- **Wails3 Alpha**: This is v3.0.0-alpha.71 - API may change
- **No Tests**: Currently no test files exist - add tests for new features
- **No Linting**: No ESLint/Prettier configured - use IDE formatting
- **Bindings**: Must regenerate after Go changes: `wails3 generate bindings`
- **Dev Port**: Default Vite port is 9245 (configurable via `VITE_PORT`)
