![bira Logo](img/bira_logo_small.png)

# bira

AI-agent optimised CLI for project management. Tracks projects, features, and tasks with clean JSON output designed for programmatic consumption.

## Install

**Prerequisites:** [Go](https://go.dev/dl/) 1.21+

### Linux / macOS
```sh
git clone https://github.com/yourorg/bira
cd bira
./install.sh
```

### Windows (PowerShell)
```powershell
git clone https://github.com/yourorg/bira
cd bira
.\install.ps1
```

Installs to `~/.bira/bin/` and adds it to your `PATH`.

---

## Quick start

```sh
cd my-project
bira init                        # create project, auto-name from directory
bira feature add "auth"          # create a feature
bira task add "login endpoint" --feature <id>
bira task add "quick fix"        # no --feature → goes to backlog
bira task done <id>
bira context --full              # project summary
```

---

## Commands

All commands accept `--json` to emit machine-readable JSON and `--project <id>` to override the project resolved from `.bira`.

### Global flags

| Flag | Description |
|------|-------------|
| `--json` | Output JSON instead of human-readable tables |
| `--project <id>` | Override project context (default: reads `.bira` walking up from cwd) |
| `--version` | Print version and exit |

---

### `bira init`

Initialize a project in the current directory.

```sh
bira init [--name <name>]
```

- Auto-detects project name from the current directory name.
- Creates `~/.bira/projects/<id>/meta.json` and an auto-created **backlog** feature.
- Writes a `.bira` config file to the current directory.
- Fails with exit code 1 if a `.bira` file already exists.

| Flag | Description |
|------|-------------|
| `--name <name>` | Override the detected project name |

---

### `bira project`

```sh
bira project list
bira project show [<id>]         # defaults to current project
bira project delete <id> --yes
```

`delete` requires `--yes` to prevent accidental data loss. Deletes the project and all its features and tasks.

---

### `bira feature`

```sh
bira feature add <name> [flags]
bira feature list [--status <status>]
bira feature show <id>
bira feature update <id> [flags]
bira feature delete <id>
bira feature note <id> <message>
```

The **backlog** feature (auto-created by `bira init`) cannot be deleted.

**Add / update flags:**

| Flag | Description |
|------|-------------|
| `--desc <text>` | Description |
| `--assign <name>` | Assigned agent or user |
| `--tags <t1,t2>` | Comma-separated tags |
| `--status <status>` | *(update only)* New status |

`bira feature note <id> <message>` appends a timestamped note to the feature. Notes are append-only.

---

### `bira task`

```sh
bira task add <title> [flags]
bira task list [--feature <id>] [--status <status>]
bira task show <id>
bira task update <id> [flags]
bira task delete <id>
bira task done <id>              # shorthand for update --status done
bira task note <id> <message>   # append a timestamped note
```

`add` without `--feature` automatically assigns the task to the project's backlog feature.

**Add / update flags:**

| Flag | Description |
|------|-------------|
| `--feature <id>` | *(add only)* Parent feature ID (default: backlog) |
| `--desc <text>` | Description |
| `--assign <name>` | Assigned agent or user |
| `--tags <t1,t2>` | Comma-separated tags |
| `--depends-on <id1,id2>` | Comma-separated dependency task IDs |
| `--criteria <text>` | Acceptance criterion — repeat flag for multiple (text may contain commas) |
| `--files <f1,f2>` | Comma-separated file references touched by this task |
| `--status <status>` | *(update only)* New status |

`bira task note <id> <message>` appends a timestamped note to the task. Notes are append-only and visible in `task show`.

---

### `bira context`

Print a snapshot of the current project state.

```sh
bira context [--full]
```

| Flag | Description |
|------|-------------|
| `--full` | Include per-feature task counts broken down by status |

Default output: project ID, name, open task count.  
`--full` output: per-feature breakdown with task counts by status.

---

## Status values

`todo` · `in-progress` · `done` · `blocked`

---

## Data storage

All data lives in `~/.bira/` — no database, no daemon.

```
~/.bira/
  projects/
    <project-id>/
      .lock               # advisory lock file (concurrent write safety)
      meta.json           # project metadata
      features/
        <feature-id>.json
      tasks/
        <task-id>.json
```

`.bira` (at your repo root) holds only `project_id` and `project_name` — a lightweight pointer into `~/.bira`.

---

## AI agent usage

bira is designed to be driven by AI agents:

- `--json` on **every** command (reads and writes) returns clean JSON to stdout.
- All errors go to **stderr** — stdout is always pure data.
- **No interactive prompts** — all input is via flags.
- Mutating commands (`add`, `update`, `delete`, `done`) echo the affected entity on stdout.
- Exit codes: `0` success · `1` general error · `2` entity not found.
- IDs are short 8-character hex strings (readable in logs and prompts).

Typical agent loop:

```sh
# Get project state
bira context --full --json

# Create work
bira feature add "payments" --json
bira task add "stripe integration" --feature <id> --assign agent-1 --json

# Update progress
bira task update <id> --status in-progress --json
bira task done <id> --json
```

---

## Development

### Build

```sh
make build          # builds ./bira.exe (Windows) or ./bira
make install        # go install with version stamp
make cross          # all 6 targets (linux/darwin/windows × amd64/arm64) → dist/
make clean
```

### Test

The test suite includes unit tests (store, config) and integration tests (command layer). All tests use isolated environments via `BIRA_HOME` — each test gets its own temp directory for data storage.

```sh
make test-local     # native go test (Windows/Linux/macOS)
make test           # run in Docker container (golang:1.26)
```

**Native test run:**
```sh
cd src && go test ./... -v -count=1
```

**Verify coverage:**

All output functions have been instrumented to accept `io.Writer` for clean testing. This enables:
- Unit tests to capture command output to buffers
- No file system pollution (each test uses `t.TempDir()` + `BIRA_HOME`)
- Sequential test execution (safe because `rootCmd` is process-wide)

**Test breakdown:**
- `internal/store` — 9 tests (storage layer, file ops, BIRA_HOME env var)
- `internal/config` — 6 tests (config discovery, project ID resolution)
- `cmd` — 13 integration tests (init, features, tasks, context, JSON output)

**Total: 28 tests, all passing.**

### Bump version

Edit [VERSION](VERSION) (single line, semver):

```
0.2.0
```

The next build will stamp `bira --version` as `0.2.0+<git-short-hash>`.

### Version output

| Build method | `bira --version` |
|---|---|
| `make build` / install scripts | `0.1.0+abc1234` |
| Plain `go build` (no ldflags) | `dev+none` |
