**Purpose**
- This file guides agentic changes in the Gentro OS repo.
- Keep edits focused on launcher-core (Rust) and launcher-ui (Svelte/Tauri).
- Avoid touching generated artifacts unless explicitly asked.

**Repo Layout**
- `launcher-core/`: Rust JSON-RPC daemon + CLI client.
- `launcher-ui/`: SvelteKit UI shell + Tauri wrapper.
- `system-services/`: placeholder for system services.
- `plan/`: architecture and planning docs.
- `scripts/`: local development helpers.

**Cursor/Copilot Rules**
- No `.cursor/rules`, `.cursorrules`, or `.github/copilot-instructions.md` found.
- If these appear later, follow them as highest-priority repo rules.

**Build And Run: Root**
- Core daemon via docker: `./scripts/dev-core`.
- Equivalent docker command: `docker compose up --build launcher-core`.
- Docker maps data/logs/socket under `.gentro/` in repo root.

**Build And Run: launcher-core (Rust)**
- Dev run: `cargo run` (runs `launcher-core`).
- Run CLI client: `cargo run --bin gentroctl -- core.status`.
- Run HTTP server by setting `GENTRO_TCP_ADDR` (see docker-compose).
- Build release: `cargo build --release`.
- Format: `cargo fmt` (rustfmt conventions).
- Lint: `cargo clippy` (not configured, optional).

**Build And Run: launcher-ui (Svelte/Tauri)**
- Install deps: `npm install` in `launcher-ui/`.
- Dev server: `npm run dev`.
- Dev with URL override: `VITE_GENTRO_RPC_URL=http://localhost:8123/rpc npm run dev`.
- Typecheck: `npm run check`.
- Watch typecheck: `npm run check:watch`.
- Build: `npm run build`.
- Preview: `npm run preview`.
- Tauri CLI passthrough: `npm run tauri <cmd>` (ex: `npm run tauri dev`).

**Tests**
- Rust tests: `cargo test` in `launcher-core/`.
- Single Rust test: `cargo test <test_name>`.
- Single Rust test in module: `cargo test module_name::test_name`.
- No JS/Svelte test runner configured; use `npm run check` for type safety.

**Docker/IPC Notes**
- Docker exposes HTTP JSON-RPC at `http://localhost:8123/rpc`.
- Host sockets cannot reach Docker sockets on macOS; use HTTP for UI dev.
- Socket path inside repo: `.gentro/run/launcher.sock`.
- Data path inside repo: `.gentro/data`.
- Log path inside repo: `.gentro/logs/gentro-core.log`.

**Rust Style (launcher-core)**
- Edition: 2024 (see `launcher-core/Cargo.toml`).
- Indentation: 4 spaces, rustfmt default.
- Imports: keep std imports grouped separately from external crates.
- Prefer `Result<T, CoreError>` with `?` for propagation.
- Use `thiserror` for error enums; avoid ad-hoc string errors.
- Map errors to JSON-RPC responses in `error_response`.
- Favor small helper functions (`ensure_dir`, `data_dir`).
- Avoid `unwrap` except in tests or when explicitly justified.
- Use `tracing` macros (`info!`, `warn!`, `error!`) for logging.
- Keep constants in `SCREAMING_SNAKE_CASE`.
- Structs/enums in `PascalCase`, functions in `snake_case`.
- Use `serde` derive for JSON payload structs.
- Keep JSON-RPC method names as string literals (`core.status`).

**Rust Error Handling**
- Convert IO/DB/JSON errors into `CoreError` variants.
- Prefer explicit error codes/messages in `error_response`.
- Use `StatusCode::BAD_REQUEST` for invalid JSON on HTTP handler.
- For Unix socket handler, reject empty payloads early.

**TypeScript/Svelte Style (launcher-ui)**
- TypeScript strict mode is enabled (`tsconfig.json`).
- Use `script lang="ts"` in Svelte components.
- Prefer `const` for static values, `let` only when reassigned.
- Use single quotes in TS and Svelte script blocks.
- Use explicit union types for UI state (`'ok' | 'warn' | 'error'`).
- Favor `as const` for literal-return helpers (`resolveRpcTarget`).
- Keep `import.meta.env` access near top for config.
- Use `void` to intentionally ignore async results in lifecycle hooks.
- Use `?` optional chaining when reading JSON-RPC response shapes.

**Svelte/CSS Conventions**
- Svelte files currently use tab indentation; keep consistent.
- Component structure: `<script>` then markup, then `<style>`.
- CSS class naming uses BEM-like `block__element--modifier`.
- Use `:global` sparingly for page-wide styles.
- Prefer scoped component styles over global styles.
- Keep UI copy short and declarative.

**Imports (Frontend)**
- Order: external packages, `$lib` aliases, then relative paths.
- Keep imports minimal; avoid unused imports (svelte-check enforces).

**Naming Conventions**
- Rust: `snake_case` functions/vars, `PascalCase` types.
- TS: `camelCase` vars/functions, `PascalCase` components/types.
- CSS: kebab-case blocks, double-underscore elements, double-dash modifiers.

**Data And Environment**
- Do not commit `.gentro/` artifacts (ignored in `.gitignore`).
- Default env vars used by core:
- `GENTRO_SOCKET_PATH`, `GENTRO_TCP_ADDR`, `GENTRO_DATA_DIR`, `GENTRO_LOG_PATH`.
- UI env vars:
- `VITE_GENTRO_RPC_URL`, `VITE_GENTRO_RPC_SOCKET` (reserved).

**Generated Artifacts**
- `launcher-ui/.svelte-kit/` is generated; do not edit manually.
- `launcher-core/target/` is build output; ignore in edits.
- Tauri icons in `launcher-ui/src-tauri/icons/` are assets.

**When Adding Features**
- Keep JSON-RPC methods in `handle_request` consistent and documented.
- Update both Unix socket and HTTP behavior when changing RPC handling.
- Extend DB schema in `init_db` carefully (migrations not present yet).
- For UI, wire RPC calls through fetch with HTTP JSON-RPC payloads.

**Docs**
- For dev environment details, consult `plan/dev-environment.md`.
- For architecture planning, reference files in `plan/`.
