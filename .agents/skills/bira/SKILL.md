---
name: bira
description: "Use when: managing project tasks, tracking features, tracking work items, running bira CLI commands, reading project status, creating tasks or features, updating task status, planning work with bira, workspace contains a .bira file, bira init, bira task, bira feature, bira context, bira project. Teaches AI agents how to operate the bira CLI tool to manage projects, features, and tasks in any repository that has been initialized with bira."
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

Out of scope:
- Contributing to bira's own Go source code
- Installing bira (see README.md)

---

## Data Model

```
Project
  └─ Feature  (named unit of work; status: todo | in-progress | done | blocked)
       └─ Task (atomic work item; status: todo | in-progress | done | blocked)
```

### Special: Backlog feature

Every project has exactly one **backlog** feature, auto-created by `bira init`:
- `IsBacklog: true` — never deletable
- Default parent for any task created without `--feature`
- Use it for ungroomed or unassigned tasks

### IDs

All IDs are **8-character lowercase hex strings** (e.g., `a3f1c9b2`). Always use the full ID
when referencing a feature or task in commands.

### Valid status values

| Value | Meaning |
|---|---|
| `todo` | Not yet started (default on creation) |
| `in-progress` | Actively being worked |
| `done` | Complete |
| `blocked` | Waiting on something external |

Any status may transition to any other status — there is no enforced state machine.

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

### 2. Get a snapshot before planning

Always run this before deciding what to work on:

```bash
bira context --full --json
```

This returns per-feature task counts broken down by status. Parse it to understand open work.

### 3. Create features for discrete deliverables

```bash
# Create a named feature for a logical unit of work
bira feature add "Add authentication" --desc "OAuth2 + JWT" --assign "agent"

# List features to get their IDs
bira feature list --json
```

Keep small or ungroomed tasks in the **backlog** — don't create a feature unless the work is
well-defined enough to group separate tasks under it.

### 4. Break features into tasks

```bash
# Task under a specific feature
bira task add "Implement login endpoint" --feature <feature-id>

# Task in the backlog (no --feature needed)
bira task add "Investigate auth library options"

# With metadata
bira task add "Write unit tests" --feature <feature-id> \
  --desc "Cover happy path and error cases" \
  --assign "agent" \
  --tags "testing,auth" \
  --depends-on "<other-task-id>"
```

### 5. Update status as you work

```bash
# Mark in-progress when starting
bira task update <id> --status in-progress

# Shorthand to mark done (preferred over update --status done)
bira task done <id>

# Mark a feature done when all its tasks are complete
bira feature update <id> --status done
```

### 6. Verify completion

```bash
# Confirm the task shows as done
bira task show <id> --json

# Check remaining open work
bira context --full --json
```

---

## Full Command Reference

### Global flags (all commands)

| Flag | Description |
|---|---|
| `--json` | Output machine-readable JSON to stdout |
| `--project <id>` | Override project context (bypasses `.bira` file lookup) |
| `--version` | Print version and exit |

**Exit codes:** `0` = success · `1` = general error · `2` = entity not found

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

# Delete (cannot delete the backlog feature)
bira feature delete <id>
```

---

### `bira task`

```bash
# Create
bira task add <title> \
  [--feature <id>]          # default: backlog
  [--desc <text>] \
  [--assign <name>] \
  [--tags <t1,t2>] \
  [--depends-on <id1,id2>]  # informational only, not validated

# Read
bira task list [--feature <id>] [--status <status>]
bira task show <id>

# Update
bira task update <id> [--status <status>] [--desc <text>] [--assign <name>] \
  [--tags <t1,t2>] [--depends-on <id1,id2>]

# Shorthand to mark done
bira task done <id>

# Delete
bira task delete <id>
```

---

### `bira context`

```bash
bira context          # summary: project ID, name, open task count
bira context --full   # per-feature breakdown with task counts by status
```

`--full` output per feature: `feature_id`, `name`, `status`, `is_backlog`, and counts for
`todo`, `in_progress`, `done`, `blocked`.

---

## JSON Output and Automation Patterns

### Always use `--json` in scripts

```bash
# Get full project state as JSON
bira context --full --json

# List open tasks across all features
bira task list --status todo --json
bira task list --status in-progress --json

# Get a specific task's details
bira task show <id> --json
```

### Parse with jq

```bash
# Count open tasks
bira context --json | jq '.open_task_count'

# Get IDs of all todo tasks in a feature
bira task list --feature <id> --status todo --json | jq '.[].id'

# Get all feature names and their statuses
bira feature list --json | jq '.[] | {id, name, status}'

# Check if any tasks are blocked
bira task list --status blocked --json | jq 'length > 0'
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
# exit 2 → task not found
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
| Treating `--depends-on` as enforced | Dependencies are informational only — bira does not block execution |
| Running `bira task list` outside a repo without `--project` | Provide `--project <id>` or `cd` into the repo first |
| Creating a feature for every single task | Small or one-off work belongs in the backlog |
| Forgetting to mark tasks `in-progress` before working | Update status before starting so `bira context` reflects real state |
