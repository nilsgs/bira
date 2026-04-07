---
name: bira
description: "Use when: managing project tasks, tracking features, tracking work items, running bira CLI commands, reading project status, creating tasks or features, updating task status, planning work with bira, workspace contains a .bira file, bira init, bira task, bira feature, bira context, bira project, bira session, claiming tasks, agent task pickup, ready tasks. Teaches AI agents how to operate the bira CLI tool to manage projects, features, and tasks in any repository that has been initialized with bira."
---

# bira

**bira** is a lightweight, file-based CLI project manager designed for AI-agent use. It tracks
**Projects → Features → Tasks** in `~/.bira/` with clean JSON output optimized for programmatic
consumption.

## Scope

Use this skill when:
- The workspace root contains a `.bira` file (bira is already initialized)
- You need to read the current work state before planning changes
- You need to create, update, or complete tasks and features
- You need a machine-readable snapshot of open work (`bira context --json`)
- You need to claim tasks for atomic agent pickup

Out of scope:
- Installing bira from source (see README.md)

> **Note:** When developing bira itself, this skill still applies — use bira to track your own
> work (features, tasks, findings). AGENTS.md defines the full workflow. The fact that you are
> editing bira's Go source does not exempt you from running `bira feature add`, creating a branch,
> or committing per-task.

---

## Data Model

```
Project
  └─ Feature  (named unit of work; status: todo | in-progress | done | blocked)
       └─ Task (atomic work item; status: todo | in-progress | done | blocked)

Session  (agent identity for claiming tasks; project-scoped)
```

### Special: Backlog feature

Every project has exactly one **backlog** feature, auto-created by `bira init`:
- `IsBacklog: true` — never deletable
- Default parent for any task created without `--feature`
- Use it for ungroomed or unassigned tasks

### IDs

All IDs are **8-character lowercase hex strings** (e.g., `a3f1c9b2`). Always use the full ID
when referencing a feature, task, or session in commands.

### Valid status values

| Value | Meaning |
|---|---|
| `todo` | Not yet started (default on creation) |
| `in-progress` | Actively being worked |
| `done` | Complete |
| `blocked` | Waiting on something external |

Any status may transition to any other status — there is no enforced state machine.  
**Exception:** `task update --status` is blocked on claimed tasks — use `task release` or `task done` instead.

---

## Recommended Workflow

### 1. Check if bira is initialized

```bash
# A .bira file at the repo root means the project is already initialized
cat .bira
```

If absent, initialize once:

```bash
bira init                   # uses current directory name as project name
bira init --name "My App"   # explicit name
```

### 2. Start a session (for agent task pickup)

Before claiming any tasks, start a session to get a stable identity:

```bash
export BIRA_SESSION=$(bira session start --label "my-agent" --json | jq -r '.id')
```

The session ID is automatically used by `task claim`, `task release`, and `task done` via `$BIRA_SESSION`.

### 3. Get a snapshot before planning

Always run this before deciding what to work on:

```bash
bira context --full --json
```

This returns per-feature task counts plus agent counters (`ready_task_count`, `claimed_task_count`, `active_session_count`).

### 4. Create features for discrete deliverables

```bash
# Create a named feature for a logical unit of work
bira feature add "Add authentication" --desc "OAuth2 + JWT" --assign "agent"

# List features to get their IDs
bira feature list --json
```

Keep small or ungroomed tasks in the **backlog** — don't create a feature unless the work is
well-defined enough to group separate tasks under it.

### 5. Break features into tasks

```bash
# Task under a specific feature
bira task add "Implement login endpoint" --feature <feature-id>

# Task in the backlog (no --feature needed)
bira task add "Investigate auth library options"

# With full metadata (required for ready_for_agent=true)
bira task add "Write unit tests" --feature <feature-id> \
  --desc "Cover happy path and error cases" \
  --assign "agent" \
  --tags "testing,auth" \
  --depends-on "<other-task-id>" \
  --criteria "All edge cases covered" \
  --criteria "Coverage >= 80%" \
  --files "src/cmd/task.go,src/internal/models/task.go"
```

`--criteria` is repeatable — each flag invocation adds one criterion. Text may contain commas.
`--files` is comma-separated and records which files the task touches.
`--depends-on` is validated on write: existence is checked, cycles are rejected.

### 6. Claim tasks for atomic agent pickup

```bash
# Find tasks ready for agent pickup
bira task list --ready --json

# Claim a task (session ID from $BIRA_SESSION)
bira task claim <task-id> --json

# Or claim even if assigned_to doesn't match your session label
bira task claim <task-id> --force --json
```

A task is `ready_for_agent=true` when it has description, at least one acceptance criterion,
a non-done feature, `status==todo`, and all dependencies are done.

### 7. Complete or release claimed tasks

```bash
# Mark done (clears the claim)
bira task done <task-id>

# Release back to the pool
bira task release <task-id> --note "context switch"

# Release to blocked
bira task release <task-id> --status blocked --note "waiting on API"
```

### 8. Verify completion

```bash
# Confirm the task shows as done
bira task show <id> --json

# Check remaining open work
bira context --full --json
```

### 9. End the session

```bash
bira session end $BIRA_SESSION
```

This releases all tasks still claimed by the session (setting them back to `todo`).

---

## Full Command Reference

### Global flags (all commands)

| Flag | Description |
|---|---|
| `--json` | Output machine-readable JSON to stdout |
| `--project <id>` | Override project context (bypasses `.bira` file lookup) |
| `--version` | Print version and exit |

**Exit codes:** `0` = success · `1` = general error · `2` = entity not found or dependency not found

All errors are written to **stderr**. Stdout is clean data when `--json` is used.

---

### `bira init`

```bash
bira init [--name <name>]
```

- Creates `~/.bira/projects/<id>/meta.json` and an auto-backlog feature
- Writes `.bira` file to the current directory
- Fails if `.bira` already exists

---

### `bira project`

```bash
bira project list                    # list all projects
bira project show [<id>]             # show current project (or by ID)
bira project delete <id> --yes       # DESTRUCTIVE: deletes all features and tasks too
```

---

### `bira feature`

```bash
# Create
bira feature add <name> [--desc <text>] [--assign <name>] [--tags <t1,t2>]

# Read
bira feature list [--status <status>]
bira feature show <id>

# Update
bira feature update <id> [--status <status>] [--desc <text>] [--assign <name>] [--tags <t1,t2>]

# Append a timestamped note (append-only)
bira feature note <id> <message>

# Delete (cannot delete the backlog feature)
# --move-tasks-to is required when the feature has tasks; target must not be done
bira feature delete <id> [--move-tasks-to <target-feature-id>]
```

---

### `bira task`

```bash
# Create
bira task add <title> \
  [--feature <id>]             # default: backlog
  [--desc <text>] \
  [--assign <name>] \
  [--tags <t1,t2>] \
  [--depends-on <id1,id2>]     # validated: existence + cycle detection
  [--criteria <text>]          # acceptance criterion; repeat for multiple
  [--files <f1,f2>]            # comma-separated file references

# Read
bira task list [--feature <id>] [--status <status>] \
  [--ready] [--assigned <name>] [--unassigned] \
  [--claimed-by <session-id>] [--unclaimed]
bira task show <id>            # includes ready_for_agent, missing_fields, blocked_reasons

# Update (replaces existing values for --criteria and --files)
# --status is blocked if the task is currently claimed
bira task update <id> [--status <status>] [--desc <text>] [--assign <name>] \
  [--tags <t1,t2>] [--depends-on <id1,id2>] \
  [--criteria <text>] [--files <f1,f2>]

# Shorthand to mark done (also clears any claim)
bira task done <id> [--session <id>]

# Append a timestamped note (append-only)
bira task note <id> <message>

# Delete (blocked if any non-done task depends on this one)
bira task delete <id>

# Claim a ready task for a session
bira task claim <id> [--session <id>] [--force]

# Release a claimed task back to the pool
bira task release <id> [--session <id>] [--status todo|blocked] [--note <text>] [--force]
```

**Task readiness** (`ready_for_agent=true`) requires all of:
- `description` is set
- at least one `acceptance_criterion`
- parent feature exists and is not `done`
- `status == todo`
- not claimed (or claim is stale / timed out)
- all `depends_on` tasks are `done`

---

### `bira session`

```bash
# Start a new session (get a stable ID for claiming tasks)
bira session start [--label <name>]        # label must be unique if provided

# List active sessions
bira session list

# Show a session and its claimed tasks (with readiness)
bira session show <id>

# End a session — releases all claimed tasks
bira session end <id> [--release-status todo|blocked]
```

Set `$BIRA_SESSION` so `--session` is implicit on `claim`, `release`, and `done`:

```bash
export BIRA_SESSION=$(bira session start --label "my-agent" --json | jq -r '.id')
```

---

### `bira context`

```bash
bira context          # summary: project ID, name, open task count
bira context --full   # per-feature breakdown + agent counters
```

`--full` top-level fields: `ready_task_count`, `claimed_task_count`, `invalid_task_count`, `active_session_count`.  
Per-feature: `feature_id`, `name`, `status`, `is_backlog`, task counts by status, and `agent` sub-object with `ready`/`claimed`/`invalid` counts.

---

## JSON Output and Automation Patterns

### Always use `--json` in scripts

```bash
# Get full project state as JSON
bira context --full --json

# Find tasks ready for agent pickup
bira task list --ready --json

# List open tasks across all features
bira task list --status todo --json
bira task list --status in-progress --json

# Get a specific task's details (includes readiness fields)
bira task show <id> --json
```

### Parse with jq

```bash
# Count tasks ready for agent pickup
bira context --full --json | jq '.ready_task_count'

# Get IDs of all ready tasks
bira task list --ready --json | jq '.[].id'

# Get a task's readiness details
bira task show <id> --json | jq '{ready_for_agent, missing_fields, blocked_reasons}'

# Get IDs of all todo tasks in a feature
bira task list --feature <id> --status todo --json | jq '.[].id'

# Get all feature names and their statuses
bira feature list --json | jq '.[] | {id, name, status}'

# Start a session and export its ID
export BIRA_SESSION=$(bira session start --label "my-agent" --json | jq -r '.id')

# List active sessions with claimed task counts
bira session list --json | jq '.[] | {id, label, claimed_task_count}'
```

### Use `--project` when outside a repo

If your shell is not inside a directory with a `.bira` file, supply the project ID explicitly:

```bash
bira task list --project <project-id> --json
```

### Exit code checking

```bash
bira task show <id> --json
# exit 0 → task exists, output is JSON
# exit 2 → task not found (or dependency not found on add/update)
# exit 1 → other error (message on stderr)
```

---

## Backlog vs. Feature: When to Use Which

| Situation | Use |
|---|---|
| Quick fix, exploration, or spike with no clear grouping | Backlog |
| Work item discovered mid-session that isn't part of current feature | Backlog |
| A well-defined deliverable with 2+ related tasks | Named feature |
| Work assigned to a specific area (e.g., "auth", "UI", "data layer") | Named feature |
| Long-lived workstream tracked across sessions | Named feature |

Keep the backlog for ungroomed work. Promote tasks to a named feature when you're ready to
commit to a plan.

---

## Common Mistakes

| Mistake | Correct approach |
|---|---|
| Using human-readable output in scripts | Always pass `--json`; table output is for display only |
| Trying to delete the backlog feature | The backlog cannot be deleted — move tasks out and leave it |
| Using an invalid status string | Only `todo`, `in-progress`, `done`, `blocked` are valid |
| Referencing a feature by name instead of ID | Always use the 8-char hex ID |
| Running `bira task list` outside a repo without `--project` | Provide `--project <id>` or `cd` into the repo first |
| Creating a feature for every single task | Small or one-off work belongs in the backlog |
| Forgetting to mark tasks `in-progress` before working | Update status before starting so `bira context` reflects real state |
| Claiming a task without a session | Start a session first: `bira session start --label "name"` |
| Trying to `task update --status` on a claimed task | Use `bira task release` or `bira task done` to change status on claimed tasks |
| Deleting a feature that has tasks without `--move-tasks-to` | Provide `--move-tasks-to <feature-id>` (target must not be done) |
| Deleting a task that other non-done tasks depend on | Remove the dependency from dependents first, then delete |
| Using a self-reference or cycle in `--depends-on` | bira validates deps on write and rejects cycles with exit 2 |
