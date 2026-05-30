# Forge CLI — Database & Project Toolkit

Forge is an opinionated command‑line toolkit for Go-era projects. From a single
binary it manages **database migrations**, **YAML seeders**, **schema
introspection / ERD / drift detection**, **Go model generation**, **project
scaffolding**, and an **external plugin system** — across **sqlite, postgres and
mysql**.

- Built in Go, no runtime dependencies.
- One DSN to configure (`FORGE_DB_DSN`); an interactive wizard builds it for you.
- Raw‑SQL migrations with up/down and batches — no ORM lock‑in.

---

## Table of contents

- [Installation](#installation)
- [Updating Forge](#updating-forge)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Command reference](#command-reference)
- [Database](#database)
  - [Migrations](#migrations)
  - [Running raw SQL (`db exec`)](#running-raw-sql-db-exec)
  - [Schema introspection & diagrams](#schema-introspection--diagrams)
  - [Schema snapshot & drift detection](#schema-snapshot--drift-detection)
  - [Generate Go models (`make:model`)](#generate-go-models-makemodel)
- [Seeders](#seeders)
- [Project scaffolding](#project-scaffolding)
- [Git integration](#git-integration-for-existing-projects)
- [Plugins](#plugins)
- [Directory structure](#directory-structure)
- [Contributing](#contributing)
- [License](#license)

---

## Installation

Install Forge **from a GitHub release (recommended)** or **from source**.

### 1. From GitHub Releases

Asset naming pattern: `forge-<os>-<arch>` — e.g. `forge-darwin-arm64`,
`forge-darwin-amd64`, `forge-linux-amd64`.

```bash
# macOS (Apple Silicon)
curl -L https://github.com/acolev/forge/releases/latest/download/forge-darwin-arm64 -o /usr/local/bin/forge
chmod +x /usr/local/bin/forge
forge --help
```

Swap the asset name for your platform (`forge-darwin-amd64`, `forge-linux-amd64`).
To avoid `sudo`, install into `~/bin` and add it to your `PATH`.

### 2. From source

```bash
git clone https://github.com/acolev/forge.git
cd forge
go mod tidy
go build -o forge ./cmd/main.go
mv forge /usr/local/bin/forge
```

---

## Updating Forge

```bash
forge upgrade           # fetch the latest release and replace the binary in place
sudo forge upgrade      # if Forge lives in a system dir like /usr/local/bin
```

`upgrade` downloads the release asset matching your OS/arch and atomically
replaces the running binary (staged next to the target, so it works across
filesystems).

---

## Quick start

```bash
# 1. Configure the database interactively (writes .env.forge)
forge config

# 2. Create and apply a migration
forge db make:sql create_table_users
#    edit database/migrations/<ts>_create_table_users.sql
forge db migrate

# 3. Generate a fixture seed straight from the table and apply it
forge seed make --from-table users --count 20
forge seed up

# 4. Inspect the schema
forge db schema:show
forge db schema:erd -o docs/erd.mmd

# 5. Generate Go structs for your app
forge db make:model -o models/models.go
```

---

## Configuration

Forge reads settings from `.env.forge` (falling back to `.env`). Values already
present in the process environment win, so you can override per shell/CI.

| Variable                | Purpose                                  | Default                      |
| ----------------------- | ---------------------------------------- | ---------------------------- |
| `FORGE_DB_DSN`          | Database connection string               | `sqlite://database/database.db` |
| `FORGE_PLUGINS_DIR`     | Local plugins directory                  | `.forge/plugins`             |
| `FORGE_MODELS_DIR`      | Default output dir for `make:model`      | `models`                     |
| `FORGE_MODELS_PACKAGE`  | Default Go package for `make:model`      | `models`                     |

### DSN formats

```env
FORGE_DB_DSN=sqlite://database/database.db
FORGE_DB_DSN=postgres://user:pass@localhost:5432/dbname
FORGE_DB_DSN=mysql://user:pass@localhost:3306/dbname
```

### Interactive wizard — `forge config`

The recommended way to set things up. It asks for the driver and connection
details step by step (host/port have skippable defaults; the password is read
with **hidden input** on a terminal), then the plugins dir and the model output
dir/package. It **pre-fills** answers from any existing `.env.forge`, only
rewrites Forge's own keys (your other variables and comments are preserved), and
can **test the connection** at the end.

```bash
forge config          # run the wizard
forge config show     # print the resolved configuration
```

### Non‑interactive defaults — `forge init` / `forge env`

```bash
forge init    # append default Forge variables to .env.forge (no prompts; good for CI)
forge env     # print the raw .env.forge contents (or resolved defaults)
```

---

## Command reference

```
forge config                 Interactive configuration wizard
  config show                Show resolved configuration
forge init                   Write default .env.forge (non-interactive)
forge env                    Print current env file

forge db make:sql [name]     Create a migration file (uses stub if prefix matches)
forge db migrate             Apply pending migrations         [--dry-run]
forge db status              Show applied / pending migrations
forge db rollback            Roll back the last batch         [--step N]
forge db reset               Roll back ALL migrations         [--force]
forge db refresh             Reset, then re-apply all         [--force]
forge db fresh               Drop ALL tables, then migrate    [--force]
forge db exec [sql]          Run raw SQL  [--file f] [--format table|json|csv] [-]
forge db schema:show         Human-readable schema overview   [--all]
forge db schema:dump         SQL DDL dump                     [-o file] [--all]
forge db schema:erd          ERD diagram   [--format mermaid|dot] [-o file] [--all]
forge db schema:snapshot     Save schema to JSON snapshot     [-o file] [--all]
forge db schema:diff         Diff live DB vs snapshot   [--from f] [--exit-code] [--all]
forge db make:model [table]  Generate Go structs  [-o file] [-p package] [--all]

forge seed make [name]       Create a seed file  [--type fixture|sql|go]
                                                 [--from-table T] [--count N]
forge seed up                Apply all pending seeders
forge seed run --only=a,b    Apply selected seeders
forge seed status            Show executed seeders
forge seed reset             Clear seeder run state (keeps data)

forge project create         Scaffold a new project (wizard or flags)
forge project git:add        Add git + remote to an existing project

forge plugins create v/n     Scaffold a plugin     [--hook <event>]
forge plugins build v/n      Build a source plugin [--global]
forge plugins install v/n    Install a local plugin globally

forge upgrade                Update Forge to the latest release
```

---

## Database

Migrations live in `./database/migrations`; applied migrations are tracked in a
`migrations` table (grouped into **batches**). Every migration file has an
`-- UP` and a `-- DOWN` section:

```sql
-- UP
CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL);

-- DOWN
DROP TABLE users;
```

### Migrations

#### Create

```bash
forge db make:sql create_table_users
# -> database/migrations/1763632453_create_table_users.sql
```

If the name starts with a known stub prefix, Forge fills the file from a
template. Built‑in stubs: `create_table`, `update_table`, `add_column`,
`drop_column`, `create_pivot_table`, `add_index`, `drop_index`. User stubs in
`database/stubs/<name>.stub.sql` take priority and may use the `{table_name}`
placeholder.

#### Apply, preview, status

```bash
forge db migrate              # apply all pending migrations (in one transaction)
forge db migrate --dry-run    # print the SQL that would run, apply nothing
forge db status               # table of applied / pending migrations
```

#### Roll back

```bash
forge db rollback             # roll back the most recent batch
forge db rollback --step 3    # roll back the last 3 batches
```

#### Destructive resets (guarded — prompt unless `--force`)

```bash
forge db reset     # run DOWN for ALL migrations
forge db refresh   # reset, then re-apply everything
forge db fresh     # DROP ALL TABLES, then re-run every migration from scratch
```

> Only migrations with a valid `-- DOWN` section can be rolled back. `reset`,
> `refresh` and `fresh` ask for confirmation; pass `--force` in scripts/CI.

### Running raw SQL (`db exec`)

Run ad‑hoc SQL against the configured database. `SELECT`‑style statements are
rendered as a table (or JSON/CSV); other statements report affected rows.

```bash
forge db exec "SELECT * FROM users"
forge db exec "INSERT INTO users (email) VALUES ('a@x.io')"
forge db exec --file ./scripts/report.sql
echo "SELECT count(*) FROM users" | forge db exec -      # read from stdin
forge db exec --format json "SELECT * FROM users"        # json | csv | table
```

### Schema introspection & diagrams

Forge reads the **live** database schema (sqlite / postgres / mysql) and renders
it several ways. Forge's own bookkeeping tables (`migrations`, `seeds`) are
hidden by default — pass `--all` to include them.

```bash
forge db schema:show                        # tables, columns, PK, FK, indexes
forge db schema:dump -o db.sql              # SQL DDL dump
forge db schema:erd                         # Mermaid erDiagram (renders on GitHub)
forge db schema:erd --format dot -o erd.dot # Graphviz DOT
```

The Mermaid output drops straight into Markdown:

```mermaid
erDiagram
    posts }o--|| users : "user_id"
```

### Schema snapshot & drift detection

Save the current schema as a JSON snapshot, then compare the live database
against it later. Useful for catching unexpected schema **drift** (e.g. a manual
change in production, or migrations that produced something unexpected).

```bash
forge db schema:snapshot                 # writes database/schema.snapshot.json
forge db schema:diff                     # human-readable diff vs the snapshot
forge db schema:diff --from other.json   # compare against a specific snapshot
forge db schema:diff --exit-code         # exit 1 if drifted (CI guard)
```

Typical CI guard (commit the snapshot to git):

```bash
forge db migrate
forge db schema:diff --exit-code   # fails the build if the live schema drifted
```

`schema:diff` currently **reports** changes (added/removed tables, added/removed/
changed columns, index and foreign‑key changes). It does not yet generate
migrations from the diff.

### Generate Go models (`make:model`)

Generate Go structs (with gorm/json tags) from existing tables. The generated
`.go` file is **plain source for your project** — Forge writes it, *your* app
compiles it (exactly like `make:sql` emits `.sql`). It is never loaded by Forge.

```bash
forge db make:model users                 # one table -> stdout
forge db make:model users -o models/user.go
forge db make:model -o models/models.go    # all tables (internal ones excluded)
forge db make:model --package entities -o models/models.go
```

When `-o`/`--package` are omitted, Forge falls back to `FORGE_MODELS_DIR` /
`FORGE_MODELS_PACKAGE` from your config. Example output:

```go
package models

import "time"

type User struct {
	ID        uint       `gorm:"column:id;primaryKey" json:"id"`
	Email     string     `gorm:"column:email;not null" json:"email"`
	IsActive  *bool      `gorm:"column:is_active" json:"is_active"`     // nullable -> pointer
	CreatedAt *time.Time `gorm:"column:created_at" json:"created_at"`
}

func (User) TableName() string { return "users" }
```

---

## Seeders

YAML seed files live in `./database/seeds`. A seed has a `type`: `fixture`
(declarative rows), `sql` (raw SQL), or `go` (a registered Go function). Applied
seeders are tracked (in a `seeds` table) and grouped into batches, like
migrations.

```bash
forge seed make users                       # fixture scaffold
forge seed make roles --type sql            # SQL seed
forge seed make bootstrap --type go         # Go seed
forge seed make --from-table users --count 20   # generate a fixture FROM a table
forge seed up                               # apply all pending seeders
forge seed run --only=users,roles           # apply selected seeders
forge seed status                           # show executed seeders
forge seed reset                            # clear run state (does NOT delete data)
```

### Generate a fixture from a table — `--from-table`

Instead of hand‑writing columns, point Forge at an existing table. It reads the
schema, maps each column to a sensible `fake:` token (by name and type), wires
foreign keys to a `$ref` on the parent, and skips auto primary keys and
`created_at/updated_at/deleted_at`:

```bash
forge seed make --from-table users --count 20
```

```yaml
name: users
type: fixture
table: users
count: 20
template:
  email: "fake:email"
  full_name: "fake:full_name"
  company_id: "ref:companies|id=1|id"   # FK -> companies.id
  is_active: "fake:bool"
```

### Fixture format

Fixtures support explicit `rows` or generated rows via `count + template`:

```yaml
name: users
type: fixture
table: users
count: 10
template:
  name: "fake:full_name"
  email: "fake:email"
  age: "fake:int:18:65"
# Optional upsert behavior:
# on_conflict: update_all        # "", "do_nothing", or "update_all"
# conflict_key: [email]          # required for update_all
# password_fields: [password]    # bcrypt-hashed before insert
```

#### Fake tokens

`fake:first_name`, `fake:last_name`, `fake:full_name`, `fake:email`,
`fake:company`, `fake:phone`, `fake:sentence`, `fake:uuid`, `fake:bool`,
`fake:datetime`, `fake:date`, `fake:int:min:max`.

#### `$ref` — reference another table's row

Resolve a value (e.g. a foreign key) from an existing parent row:

```yaml
template:
  # shortcut form: ref:<table>|<col>=<value>|<select-column>
  company_id: "ref:companies|name=Acme|id"
  # object form (supports defaults / nested refs):
  owner_id:
    $ref:
      table: users
      where: { email: "admin@x.io" }
      select: id
```

> Fixture seeding is cross‑driver: on sqlite/mysql, map/array values are stored
> as JSON text and upserts use dialect‑appropriate SQL. JSON casts and `bytea`
> handling are Postgres‑specific.

---

## Project scaffolding

Bootstrap new projects with `forge project create` — interactively or via flags.

```bash
forge project create                         # interactive wizard
forge project create --lang go   --name my-api --dir ./services/my-api --git-init
forge project create --lang node --name my-app
forge project create --lang ts   --name my-ts-app
forge project create --lang empty --name sandbox
forge project create --from https://github.com/user/template.git --name booking-api
```

In template mode (`--from`) Forge clones the repo into a temp dir, strips its
`.git`, copies files into your target directory, and (with `--git-init`)
initializes a fresh repository.

---

## Git integration for existing projects

```bash
forge project git:add
forge project git:add --dir ./services/api --remote git@github.com:you/api.git
```

Without flags it uses the current directory, initializes git if needed, runs
`git add .`, creates an initial commit if there are none, then asks for a remote
URL (added as `origin`) and whether to push.

- `--dir` — project directory (default: current directory).
- `--remote` — remote URL for `origin` (asked interactively if omitted).

---

## Plugins

Forge runs **external plugins** as subprocesses (polyglot: Go, Node, Python),
discovered from a `plugin.json` manifest.

- Local plugins: `.forge/plugins` (default for `create`/`build`).
- Global plugins: `~/.forge/plugins`.

```bash
forge plugins create bookly/migrate              # scaffold a Go plugin
forge plugins create bookly/migrate --hook db.migrate.before
forge plugins build bookly/migrate               # build (add --global for ~/.forge)
forge plugins install bookly/migrate             # install a local plugin globally
```

`plugins create` writes:

```text
.forge/plugins/bookly/migrate/plugin.json
.forge/plugins/bookly/migrate/src/go.mod
.forge/plugins/bookly/migrate/src/main.go
```

### Source‑based Go plugins

A manifest declaring Go source is auto‑built (platform‑specific binary) before
execution:

```json
{ "lang": "go", "entry": "migrate", "source": "src" }
```

### Hooks

Plugins can subscribe to lifecycle events via their manifest:

```json
{ "hooks": { "db.migrate.before": { "command": "db-migrate-before" } } }
```

Available events: `db.migrate.before`, `db.migrate.after`,
`project.create.before`, `project.create.after`.

> Note: hook payloads are not forwarded to plugins yet.

---

## Directory structure

A Forge‑managed project typically has:

```text
.env.forge                 # Forge configuration
database/
  migrations/              # .sql migrations (-- UP / -- DOWN)
  seeds/                   # .yaml seeders
  stubs/                   # optional user migration stubs
  schema.snapshot.json     # optional schema snapshot (for schema:diff)
.forge/plugins/            # local plugins
```

Forge's own source layout:

- `cmd/main.go` — CLI entry point and root commands (`config`, `init`, `env`, `upgrade`).
- `internal/config/` — settings, DSN build/parse, the interactive wizard.
- `internal/database/` — GORM bootstrap and the `db exec` engine.
- `internal/migrations/` — migration generate/apply/rollback and the `db` command group.
- `internal/schema/` — schema introspection, dump/ERD, snapshot/diff, model generation.
- `internal/seeders/` — YAML seeder engine (sql/fixture/go, fakers, `$ref`).
- `internal/project/` — project scaffolding and git helpers.
- `internal/plugins/` — plugin discovery and execution.
- `internal/selfupdate/` — `upgrade` logic.
- `internal/hooks/` — in‑process hook bus.

---

## Contributing

```bash
git checkout -b feature/my-feature
# ... changes ...
go test ./...
go build ./cmd/main.go
git commit -am "Add my feature"
git push origin feature/my-feature
```

Then open a Pull Request.

---

## License

This project is licensed under the **MIT License**. See the `LICENSE` file for details.
