# AGENTS.md

## Project

Forge is a Go CLI for:
- SQL migrations
- YAML/Go/SQL seeders
- project scaffolding
- git initialization helpers
- self-update from GitHub releases
- external plugin execution via `.forge/plugins/**/plugin.json`

Entrypoint:
- `cmd/main.go`

Module:
- `module forge`
- Go version in `go.mod`: `1.23.1`

## Repository Layout

- `/Users/attar/GolandProjects/NinjaCode/forge/cmd/main.go`
- `cmd/main.go`
  Main Cobra CLI, root commands, plugin manager bootstrap.
- `internal/database/database.go`
  DB bootstrap through GORM. Supports `sqlite`, `mysql`, `postgres`.
- `internal/migrations/`
  SQL migration generation, apply, rollback, stub templates.
- `internal/seeders/`
  YAML seeder engine with SQL, fixture, and Go seed execution.
- `internal/project/`
  Project scaffolding, git helpers, template URL resolution, interactive wizard.
- `internal/plugins/`
  Plugin discovery and execution layer.
- `internal/selfupdate/`
  GitHub release lookup and binary replacement.
- `internal/hooks/hooks.go`
  In-process hook bus used by project/migration/plugin integration.
- `forge-project/`
  Example/test workspace used during local development. Contains `.env`, `database/`, and local `.forge/plugins`.

## CLI Surface

Root commands implemented in code:
- `forge init`
- `forge env`
- `forge self-update`
- `forge db make:sql`
- `forge db migrate`
- `forge db rollback`
- `forge seed up`
- `forge seed run`
- `forge seed status`
- `forge seed reset`
- `forge project create`
- `forge project git:add`
- plugin commands registered dynamically from manifests

## Build And Release

Primary release script:
- `scripts/release-build.sh`

GitHub Actions release workflow:
- `.github/workflows/release.yml`

Current release assets built by script:
- `dist/forge-linux-amd64`
- `dist/forge-darwin-amd64`
- `dist/forge-darwin-arm64`

Notes:
- `Makefile` is not the source of truth for plugins. It still refers to legacy `plugins/*` Go plugin builds.
- Real plugin runtime is manifest-driven and file-based under `.forge/plugins`.

## Plugin System

Actual runtime behavior:
- scan local: `<project>/.forge/plugins/**/plugin.json`
- scan global: `~/.forge/plugins/**/plugin.json`
- load manifest
- execute entry as:
  - binary directly by default
  - `node <entry>` if `lang == "node"`
  - `python <entry>` if `lang == "python"`

Protocol:
- stdin: JSON request
- stdout: JSON response
- stderr: treated as diagnostic output only

Important limitation:
- hook payloads are currently dropped before being passed to plugins to avoid JSON serialization issues

Local example plugin source:
- `forge-project/.forge/plugins/@forge/build-go-plug/src/main.go`

## Database Behavior

Environment is loaded from `.env` via `godotenv.Load()`.

Expected variables:
- `DB_DRIVER`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`

SQLite path is hardcoded:
- `database/database.db`

Migration metadata table:
- `migrations`

Seeder metadata table:
- `seeds`

## Migrations

Files live in:
- `./database/migrations`

Format:
```sql
-- UP
...
-- DOWN
...
```

Important current behavior:
- pending migrations are sorted lexicographically by filename
- migration apply now runs in a single DB transaction for the whole batch
- each applied file is recorded in `migrations`
- rollback runs the highest batch in reverse record order

Files:
- `internal/migrations/migrations.go`
- `internal/migrations/templates.go`
- `internal/migrations/commads.go`

## Seeders

Seed directory:
- `./database/seeds`

Supported seeder types:
- `sql`
- `fixture`
- `go`

Capabilities:
- per-seed transaction for SQL and Go seeders
- fixture insert/upsert
- password hashing
- `$ref` resolution
- PostgreSQL JSON/JSONB/BYTEA normalization helpers

Files:
- `internal/seeders/runner.go`
- `internal/seeders/types.go`
- `internal/seeders/refs.go`
- `internal/seeders/conflict.go`

## Project Scaffolding

Supported in code:
- `go`
- `node`
- `js`
- `ts`
- `empty`
- `none`

Git template sources supported:
- GitHub
- GitLab
- Bitbucket

Files:
- `internal/project/commads.go`
- `internal/project/init.go`
- `internal/project/new.go`
- `internal/project/git.go`

## Known Issues And Mismatches

High priority:
- `seed run` declares `only` variable but never registers a `--only` flag, so targeted seed execution is effectively unusable.
- `db rollback` ignores `database.InitDB()` error and may pass a nil DB into rollback logic.
- self-update aborts when asset selection falls back, because `selectAsset()` returns both an asset and an error and caller treats any error as fatal.
- plugin docs/build pipeline are inconsistent with runtime: `Makefile` and `scripts/build-plugins.sh` assume Go `buildmode=plugin`, while runtime expects manifest-based external executables/scripts.

Medium priority:
- README claims `forge env` shows effective environment, but implementation only prints `.env`.
- README and help mention `vue`, but scaffolding does not implement it.
- hook payload is intentionally dropped before plugin execution, which makes event hooks much less useful than the API shape suggests.
- self-update has no checksum/signature verification.

Operational:
- local `go test` currently depends on a consistent Go toolchain and build cache; if tests fail with `compile: version ... does not match go tool version ...`, fix the host Go installation before trusting test results.

## Validation Commands

Useful commands for future work:
```bash
go build -o forge ./cmd/main.go
go test ./...
go test ./internal/migrations
go test ./internal/seeders
./scripts/release-build.sh
```

If sandboxing blocks `go test`, rerun with permission to use the host Go build cache.

## Working Rules For Future Agents

- Read code before trusting `ReadMe.md`; documentation is partially stale.
- Treat `forge-project/` as a disposable local workspace, not core source.
- Do not rely on `Makefile` for plugin behavior without reconciling it with `internal/plugins`.
- Preserve Cobra command names and current output format unless intentionally changing CLI contract.
- When touching migrations or seeders, verify behavior on SQLite and consider PostgreSQL-specific code paths separately.
- When touching self-update, treat release asset naming and replacement semantics as cross-platform concerns.
- Avoid reverting unrelated user changes. Current worktree may already contain edits outside your task.
