![bira Logo](img/bira_logo_small.png)

# bira

AI-agent optimised CLI for project management. Tracks projects, features, and tasks with clean JSON output designed for programmatic consumption.

## Install

**Prerequisites:** [Go](https://go.dev/dl/) 1.21+

### Linux / macOS
```sh
git clone https://github.com/nilsgs/bira
cd bira
./install.sh
```

### Windows (PowerShell)
```powershell
git clone https://github.com/nilsgs/bira
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
bira task add "login endpoint" --feature <id> --desc "..." --criteria "..."
bira task add "quick fix"        # no --feature → goes to backlog
bira task done <id>
bira context --full              # project summary
```

**Agent session workflow:**

```sh
# Start a session (gives you a stable ID for claiming tasks)
export BIRA_SESSION=$(bira session start --label "my-agent" --json | jq -r '.id')

# Find tasks ready for pickup
bira task list --ready --json

# Claim a task atomically
bira task claim <task-id>        # uses $BIRA_SESSION automatically

# Work, then mark done
bira task done <task-id>

# Or release back to the pool
bira task release <task-id> --note "blocked by upstream change"

# End the session (releases any remaining claimed tasks)
bira session end $BIRA_SESSION
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
bira feature delete <id> [--move-tasks-to <feature-id>]
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

**Delete flags:**

| Flag | Description |
|------|-------------|
| `--move-tasks-to <id>` | Reassign all tasks to this feature before deleting (required when tasks exist; target feature must not be `done`) |

`bira feature note <id> <message>` appends a timestamped note to the feature. Notes are append-only.

---

### `bira task`

```sh
bira task add <title> [flags]
bira task list [--feature <id>] [--status <status>] [--ready] [--assigned <name>] [--unclaimed]
bira task show <id>
bira task update <id> [flags]
bira task delete <id>
bira task done <id> [--session <id>]   # shorthand for update --status done
bira task note <id> <message>          # append a timestamped note
bira task claim <id> [--session <id>] [--force]
bira task release <id> [--session <id>] [--status todo|blocked] [--note <text>] [--force]
```

`add` without `--feature` automatically assigns the task to the project's backlog feature.

`--session` defaults to the `$BIRA_SESSION` environment variable. Set it once at session start.

**Add / update flags:**

| Flag | Description |
|------|-------------|
| `--feature <id>` | *(add only)* Parent feature ID (default: backlog) |
| `--desc <text>` | Description |
| `--assign <name>` | Assigned agent or user |
| `--tags <t1,t2>` | Comma-separated tags |
| `--depends-on <id1,id2>` | Dependency task IDs — validated on write (existence + cycle detection) |
| `--criteria <text>` | Acceptance criterion — repeat flag for multiple (text may contain commas) |
| `--files <f1,f2>` | Comma-separated file references touched by this task |
| `--status <status>` | *(update only)* New status — blocked if task is currently claimed |

**List filter flags:**

| Flag | Description |
|------|-------------|
| `--ready` | Only tasks with `ready_for_agent=true` |
| `--assigned <name>` | Filter by `assigned_to` value |
| `--unassigned` | Only tasks with no `assigned_to` |
| `--claimed-by <session-id>` | Only tasks claimed by this session |
| `--unclaimed` | Only tasks with no active claim |

**Claim / release flags:**

| Flag | Description |
|------|-------------|
| `--session <id>` | Session ID to act as (default: `$BIRA_SESSION`) |
| `--force` | *(claim)* Bypass `assigned_to` check · *(release)* Release even if owned by another session |
| `--status` | *(release)* Status to set after release: `todo` (default) or `blocked` |
| `--note <text>` | *(release)* Append a note on release |

**Task readiness** (`ready_for_agent=true`) requires all of: description set, at least one acceptance criterion, feature exists and is not done, status is `todo`, not claimed (or claim is stale), all dependencies done.

`bira task note <id> <message>` appends a timestamped note to the task. Notes are append-only and visible in `task show`.

---

### `bira context`

Print a snapshot of the current project state.

```sh
bira context [--full]
```

| Flag | Description |
|------|-------------|
| `--full` | Include per-feature task counts, agent readiness counters, and active session count |

Default output: project ID, name, open task count.  
`--full` output: per-feature breakdown with task counts by status plus top-level agent fields: `ready_task_count`, `claimed_task_count`, `invalid_task_count`, `active_session_count`.

---

### `bira session`

Manage agent sessions. A session is a stable identity used for claiming tasks.

```sh
bira session start [--label <name>]   # create a session, print its ID
bira session list                     # list active sessions + claimed task count
bira session show <id>                # show session details and its claimed tasks
bira session end <id> [--release-status todo|blocked]   # end session, release claimed tasks
```

Set the session ID in your environment so `--session` is implicit:

```sh
export BIRA_SESSION=$(bira session start --label "my-agent" --json | jq -r '.id')
```

The default claim timeout is **24 hours** (configurable via `claim_timeout_minutes` in `.bira`). A task whose claim has timed out is treated as unclaimed and becomes available for pickup again.

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
      sessions/
        <session-id>.json
```

`.bira` (at your repo root) holds `project_id`, `project_name`, and optionally `claim_timeout_minutes` — a lightweight pointer into `~/.bira`.

**`.bira` example with custom claim timeout:**
```json
{
  "project_id": "a3f1c9b2",
  "project_name": "my-project",
  "claim_timeout_minutes": 480
}
```

---

## AI agent usage

bira is designed to be driven by AI agents:

- `--json` on **every** command returns clean JSON to stdout.
- All errors go to **stderr** — stdout is always pure data.
- **No interactive prompts** — all input is via flags.
- Mutating commands echo the affected entity on stdout.
- Exit codes: `0` success · `1` general error · `2` entity not found or missing dependency.
- IDs are short 8-character hex strings (readable in logs and prompts).
- `$BIRA_SESSION` sets the default session ID for `claim`, `release`, and `done`.

**Typical multi-agent loop:**

```sh
# Bootstrap: start a session
export BIRA_SESSION=$(bira session start --label "agent-1" --json | jq -r '.id')

# Discover ready work
bira task list --ready --json

# Claim atomically (fails if another agent already claimed it)
bira task claim <task-id> --json

# Work on the task...

# Mark done (clears the claim)
bira task done <task-id> --json

# Or release on failure / context switch
bira task release <task-id> --status blocked --note "hit upstream issue" --json

# Clean up when done for the day
bira session end $BIRA_SESSION --json
```

**Checking readiness in scripts:**

```sh
# Find all ready tasks assigned to this agent
bira task list --ready --assigned agent-1 --json

# Get readiness details for a specific task
bira task show <id> --json | jq '{ready: .ready_for_agent, missing: .missing_fields, blocked: .blocked_reasons}'

# Check agent counters
bira context --full --json | jq '{ready: .ready_task_count, claimed: .claimed_task_count, sessions: .active_session_count}'
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
- `cmd` — 33 integration tests (init, features, tasks, context, JSON output, sessions, claim/release, dep validation, readiness)

**Total: 48 tests, all passing.**

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
