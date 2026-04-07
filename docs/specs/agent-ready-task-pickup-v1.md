# Agent-Ready Task Pickup v1

## Summary

This spec defines the minimum bira changes required for a separate agent to:

1. discover work that is actually actionable,
2. inspect enough context to decide whether to take it,
3. claim it without racing other agents,
4. complete or release it cleanly, and
5. trust that bira is showing the full project state instead of a silent partial view.

v1 stays file-based and local-first. It does not introduce a scheduler, priorities, or remote coordination.

## Problems to Solve

Current bira is a good storage and transport layer, but it does not yet provide workflow semantics for agent pickup:

- tasks can exist with only a title and still look equivalent to fully specified work,
- dependencies are informational only,
- there is no atomic claim operation,
- list/context commands can silently skip unreadable entity files,
- feature and task deletion can leave dangling references,
- agents must reimplement readiness and claim policy outside bira.

## Goals

- Make task readiness explicit and machine-readable.
- Keep grooming permissive: humans may still create incomplete tasks.
- Add an atomic claim flow using the existing project lock.
- Make task handoff information visible in `task show --json` and `task list --json`.
- Fail loud on corrupted or incomplete state instead of silently hiding work.
- Preserve backward compatibility for existing stored projects where practical.

## Non-Goals

- No automatic task prioritization.
- No cross-machine distributed locking beyond the current filesystem lock model.
- No interactive prompts.
- No replacement of `assigned_to` with a richer planning system.

## Data Model Changes

### Stored task fields

Add these fields to the persisted task model:

```json
{
  "claimed_by": "agent-2",
  "claimed_at": "2026-04-07T10:30:00Z"
}
```

Rules:

- `claimed_by` is the runtime owner of active execution.
- `claimed_at` is set when the task is successfully claimed.
- `assigned_to` remains planning metadata. It is not removed or renamed.
- existing task files remain valid; missing `claimed_by` and `claimed_at` mean unclaimed.

### Derived task fields in JSON output

Do not persist these. Compute them on read and include them in JSON output for `task show`, `task list`, and `context --full`.

```json
{
  "ready_for_agent": true,
  "missing_fields": [],
  "blocked_reasons": [],
  "dependency_summary": {
    "total": 2,
    "done": 2,
    "missing": 0,
    "not_done": 0
  }
}
```

Definitions:

- `ready_for_agent` is `true` only when:
  - `status == "todo"`
  - `claimed_by` is empty
  - `description` is non-empty
  - `acceptance_criteria` has at least one entry
  - parent feature exists
  - parent feature is not `done`
  - every `depends_on` task exists and has `status == "done"`
- `missing_fields` contains machine-readable reasons such as:
  - `missing_description`
  - `missing_acceptance_criteria`
  - `missing_feature`
- `blocked_reasons` contains readiness blockers such as:
  - `feature_done`
  - `dependency_missing:<id>`
  - `dependency_not_done:<id>`
  - `already_claimed`
  - `status_not_todo`

`task show --json` must also include resolved dependency entries:

```json
{
  "dependencies": [
    { "id": "aaaabbbb", "exists": true, "status": "done", "title": "Write parser tests" }
  ]
}
```

If a dependency file is missing, return:

```json
{ "id": "aaaabbbb", "exists": false }
```

## Command Changes

### `bira task list`

Add filters:

- `--ready`
- `--assigned <name>`
- `--unassigned`
- `--claimed-by <name>`
- `--unclaimed`

Behavior:

- filters compose with existing `--feature` and `--status`
- `--ready` returns only tasks where `ready_for_agent == true`
- `--assigned` filters on `assigned_to`
- `--claimed-by` filters on `claimed_by`
- `--unassigned` and `--unclaimed` match empty values only
- human-readable output adds `READY` and `CLAIMED BY` columns
- JSON output includes the derived readiness fields

Ordering:

- sort deterministically by `created_at ASC`, then `id ASC`
- apply the same ordering in `task list --json` and `feature/task summaries` returned from `context --full --json`

### `bira task show`

JSON output must include:

- stored task fields,
- resolved parent feature summary:
  - `feature: { "id": "...", "name": "...", "status": "..." }`
- readiness fields,
- resolved dependencies array.

Human-readable output adds:

- `Claimed by`
- `Claimed at`
- `Ready for agent`
- `Missing fields`
- `Blocked reasons`

### `bira task claim`

Add a new command:

```bash
bira task claim <id> --agent <name> [--force]
```

Behavior:

- acquires the project lock before load-check-update-save
- fails with exit code `1` unless all claim preconditions pass
- preconditions without `--force`:
  - task exists
  - `ready_for_agent == true`
  - `claimed_by` is empty
  - `assigned_to` is empty or equals `--agent`
- on success:
  - set `claimed_by = <agent>`
  - set `claimed_at = now`
  - set `status = "in-progress"`
  - update `updated_at`
- `--force` bypasses the `assigned_to` check only
- `--force` does not bypass missing description, missing criteria, missing feature, dependency failures, or an existing claim
- JSON output returns the updated task including readiness metadata after the change

### `bira task release`

Add a new command:

```bash
bira task release <id> --agent <name> [--status todo|blocked] [--note <message>] [--force]
```

Behavior:

- default release status is `todo`
- only the current `claimed_by` agent may release unless `--force` is supplied
- on success:
  - clear `claimed_by`
  - clear `claimed_at`
  - set `status` to the requested status
  - append a note if `--note` is provided
  - update `updated_at`
- `--status done` is invalid; completion remains `task done`

### `bira task done`

Change existing behavior:

- if the task is claimed, `task done` clears `claimed_by` and `claimed_at`
- if `--agent <name>` is supplied and the task is claimed by another agent, fail with exit code `1`

New form:

```bash
bira task done <id> [--agent <name>]
```

If `--agent` is omitted, preserve current permissive behavior.

### `bira context --full`

Keep the current top-level fields and per-feature `tasks` status counts.

Add:

```json
{
  "ready_task_count": 4,
  "claimed_task_count": 2,
  "invalid_task_count": 3,
  "features": [
    {
      "id": "abcd1234",
      "name": "auth",
      "status": "in-progress",
      "is_backlog": false,
      "tasks": { "todo": 3, "in-progress": 2, "done": 1 },
      "agent": {
        "ready": 2,
        "claimed": 1,
        "invalid": 1
      }
    }
  ]
}
```

Definitions:

- `ready_task_count`: total tasks where `ready_for_agent == true`
- `claimed_task_count`: total tasks with non-empty `claimed_by` and `status == "in-progress"`
- `invalid_task_count`: total open tasks that are not ready because they fail task-spec completeness or referential checks

## Integrity and Validation Rules

### Read commands must fail loud

Change `loadAllTasks` and `loadAllFeatures` behavior:

- if any entity file cannot be read or parsed, abort the command with exit code `1`
- stderr must identify the entity file path and parse failure
- do not silently skip unreadable entities

Commands affected:

- `feature list`
- `task list`
- `context`
- any helper flow that currently scans all entities to compute results

### Dependency validation on write

On `task add` and `task update`:

- every `depends_on` ID must exist in the same project
- self-dependency is invalid
- duplicate dependency IDs are invalid

Return exit code `2` for missing dependency tasks and exit code `1` for malformed dependency sets.

### Feature deletion

Change `feature delete <id>` behavior:

- if any tasks still reference the feature, fail with exit code `1`
- add `--move-tasks-to <feature-id>` to support explicit migration during delete
- `--move-tasks-to` may target the backlog feature or another non-deleted feature in the same project
- migration happens under the same project lock before deleting the feature file

### Task deletion

Change `task delete <id>` behavior:

- if any non-done task depends on the target task, fail with exit code `1`
- error message must list the dependent task IDs

## Output and Error Semantics

- Keep the existing rule that JSON data is written to stdout and errors to stderr.
- Keep existing exit codes:
  - `0` success
  - `1` general validation or workflow error
  - `2` entity not found
- Claim/release conflicts use exit code `1`, not `2`.
- Add tests for all new command and validation branches.

## Backward Compatibility

- Existing project and task JSON files remain readable without migration.
- Existing commands remain valid unless they now violate a new integrity rule.
- Existing automation using `task list --json` continues to work because new JSON fields are additive.
- `context --full --json` remains backward compatible by preserving existing fields and appending new counters.

## Example Agent Workflow

```bash
# Discover actionable work
bira task list --ready --unclaimed --json

# Inspect one task deeply
bira task show 1a2b3c4d --json

# Atomically take ownership
bira task claim 1a2b3c4d --agent agent-2 --json

# Record a blocker and release the task
bira task release 1a2b3c4d --agent agent-2 --status blocked --note "Waiting on API schema" --json

# Reclaim later and complete it
bira task claim 1a2b3c4d --agent agent-2 --json
bira task done 1a2b3c4d --agent agent-2 --json
```

## Implementation Notes

- compute readiness in a shared helper so `task list`, `task show`, `task claim`, and `context` use identical logic
- keep write paths lock-protected using the current per-project advisory lock
- add deterministic sorting before rendering output
- do not reuse the silent-skip load helpers; replace them with load helpers that return aggregated errors

## Acceptance Tests

Add command-layer tests for:

- `task show --json` returns readiness fields and resolved dependencies
- `task list --ready --json` returns only fully specified unclaimed todo tasks with all dependencies done
- `task claim` sets `claimed_by`, `claimed_at`, and `status=in-progress`
- `task claim` fails when assigned to another agent without `--force`
- `task claim` fails when dependencies are incomplete
- `task release` clears claim metadata and appends a note when requested
- `task done --agent` fails for the wrong agent and clears claim metadata for the right agent
- `feature delete` fails when child tasks exist unless `--move-tasks-to` is supplied
- `task delete` fails when active dependents exist
- `context --full --json` includes ready, claimed, and invalid counters
- malformed task or feature JSON causes `task list` or `context` to fail instead of silently omitting data

## Definition of Done

This spec is complete when bira can be used by an external agent with this loop:

1. list ready work,
2. inspect one task,
3. claim it atomically,
4. trust that dependencies and references are valid,
5. release or finish it without leaving stale ownership,
6. and detect corrupted project state as an error instead of a partial success.
