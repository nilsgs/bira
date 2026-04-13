# Bira — Redesign Plan

## Problem Statement

The current bira model (Projects → Features → Tasks, with multi-agent session/claim/release) was designed for a world where the AI agent needs a managed todo list. Modern agents (Copilot, Claude, Codex) maintain their own internal task lists during implementation — bira duplicating that is unnecessary friction. The real value bira can offer is:

1. **Lightweight capture** — a quick way to record ideas, bugs, and proposals without context-switching.
2. **Loose coupling between planning and implementation** — bira owns the "what and why", the agent decides "how".
3. **Cross-repo coordination** — filing a bug against another project without leaving the current context.
4. **Triage** — a structured way to prioritise and enrich captured items so agents can confidently pick up the right work.

---

## Role Boundary

**bira is a support tool. It does not fill itself with content.**

The responsibility for creating well-formed, actionable bira records is shared between humans and agents in a collaborative loop:

- **Humans** — capture quick ideas, file rough bug reports, review and approve triage suggestions, decide priorities.
- **Agents** — enrich descriptions, propose triage (priority, criticality, impact/complexity), write plan sidecars, file bugs against other tools, implement features.

The typical flow is **iterative**: a human captures something raw, an agent refines it and proposes triage, the human reviews and adjusts, back and forth until the item is ready. Neither party owns the process end-to-end.

bira's job is to provide a clean, reliable CLI that makes this cooperative loop frictionless. A well-maintained `SKILL.md` (see below) is therefore as important as the CLI itself — it is the interface contract that lets any agent participate in the loop correctly without guessing.

---

## Core Design Principles

- **bira tracks intent, not execution.** No tasks. Agents manage their own implementation steps.
- **Three first-class entity types**: Idea, Bug, Feature — each with its own lifecycle and triage fields.
- **Status-driven workflow** — agents advance items through their lifecycle by updating status. No session or claim primitives in v1; one agent at a time is acceptable.
- **Storage stays local** (`~/.bira`), but the design should not preclude a future git-like sync mechanism.
- **Markdown plan sidecars** on all three entity types — free-form, writable by human or agent.
- **All design invariants from today are preserved**: `--json`, stderr for errors, exit codes, no interactive prompts, empty collections as `[]`.

## Agent Interfaces

bira exposes two complementary agent interfaces that share the same underlying store:

### 1. CLI (primary — humans & shell-based agents)
The `bira` binary is the canonical interface. Designed for humans, scripts, and agents that can invoke shell commands. All commands support `--json` for machine-readable output.

### 2. MCP Server (IDE-embedded agents — Copilot, Cursor, Claude Desktop, etc.)
`bira mcp` starts a [Model Context Protocol](https://modelcontextprotocol.io) server over **stdio**. 

**How it runs:** There is no hosting. The IDE extension (Copilot in VS Code, Cursor, Claude Desktop) spawns `bira mcp` as a local process on demand and communicates over stdin/stdout. The process exits when the IDE shuts it down. No port, no daemon, no server to manage.

**Is a subcommand idiomatic?** Yes, for a local file-based tool like bira a subcommand is the right call: it produces a single binary to install and ship, the MCP server shares 100% of the same store code as the CLI, and the client config is simply `"command": "bira", "args": ["mcp"]`. Separate MCP binaries (e.g. `bira-mcp`) are common in the ecosystem but add distribution complexity without benefit here.

**Implementation library:** [`github.com/mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go) — the de-facto standard Go MCP library. `server.ServeStdio(s)` blocks and handles the protocol loop.

The MCP server wraps the same store as the CLI — data written via MCP is immediately visible via `bira` CLI and vice versa.

**MCP tools** mirror the CLI commands as structured functions, e.g.:
- `create_idea`, `list_ideas`, `triage_idea`, `promote_idea`, `get_idea_plan`, `set_idea_plan`
- `create_bug`, `list_bugs`, `triage_bug`, `start_bug`, `done_bug`, `get_bug_plan`, `set_bug_plan`
- `create_feature`, `list_features`, `triage_feature`, `start_feature`, `done_feature`, `get_feature_plan`, `set_feature_plan`
- `list_projects`, `get_context`

**SKILL.md** covers both interfaces — CLI recipes for shell agents, and MCP tool names/parameters for IDE-embedded agents.

**MCP server config example** (VS Code / Claude Desktop `mcp.json`):
```json
{
  "mcpServers": {
    "bira": {
      "command": "bira",
      "args": ["mcp"]
    }
  }
}
```

> The IDE spawns `bira mcp` as a child process. No port, no daemon, no setup beyond the config file.

---



### Idea
Represents a raw, unvalidated thought captured quickly. No required fields beyond a title.

```
inbox → triaged → promoted → (becomes a Feature)
                → rejected
```

**Fields**: id, project_id, title, description, tags, priority (after triage), notes[], plan_path (optional .md sidecar), status, created_at, updated_at

**Triage fields**: priority (`low|medium|high`), tags

**Key commands**:
- `bira idea add <title> [--desc <text>] [--tags t1,t2]`
- `bira idea list [--status <status>]`
- `bira idea show <id>`
- `bira idea triage <id> --priority <low|medium|high> [--tags t1,t2]`
- `bira idea promote <id>` — converts idea to Feature (status → `proposed`); plan sidecar carried over if present
- `bira idea plan <id> [--edit | --stdin]` — read/write Markdown sidecar
- `bira idea note <id> <message>`
- `bira idea delete <id>`

---

### Bug
Represents a defect filed against a specific project, potentially by an agent working in a different repo. Supports multiline descriptions (via stdin). Criticality is set during triage.

```
open → triaged → in-progress → fixed
                              → wont-fix
```

**Fields**: id, project_id, title, description, criticality (after triage: `low|medium|high|critical`), reported_by (free-text, optional), tags, notes[], plan_path (optional .md sidecar), status, created_at, updated_at

**Key commands**:
- `bira bug create [-p <id|slug>] <title> [--desc <text>] [--reported-by <label>]`
  - `--desc -` reads description from stdin (multiline support)
- `bira bug list [-p <id>] [--status <status>] [--criticality <level>]`
- `bira bug show <id>`
- `bira bug triage <id> --criticality <low|medium|high|critical> [--tags t1,t2]`
- `bira bug start <id>` — sets status to `in-progress`
- `bira bug done <id>` — marks fixed
- `bira bug wont-fix <id> [--note <text>]` — closes without fixing
- `bira bug plan <id> [--edit | --stdin]` — read/write Markdown investigation/fix-plan sidecar
- `bira bug note <id> <message>`
- `bira bug delete <id>`

---

### Feature
A well-defined unit of work, typically promoted from an Idea or added directly. Has a planning artifact (JSON metadata + Markdown implementation plan sidecar).

```
proposed → triaged → in-progress → done
                                 → rejected
```

**Fields**: id, project_id, title, description, impact (`low|medium|high`), complexity (`low|medium|high`), tags, notes[], plan_path (relative path to .md sidecar), status, promoted_from (idea_id, optional), created_at, updated_at

**Key commands**:
- `bira feature add <title> [--desc <text>] [--tags t1,t2]`
- `bira feature list [--status <status>]`
- `bira feature show <id>`
- `bira feature triage <id> --impact <low|medium|high> --complexity <low|medium|high> [--tags t1,t2]`
- `bira feature plan <id> [--edit | --stdin]`
  - No flags: prints the current plan (or empty state prompt)
  - `--edit`: opens sidecar in `$EDITOR`
  - `--stdin`: reads plan content from stdin (agent-friendly)
- `bira feature start <id>` — sets status to `in-progress`
- `bira feature done <id>`
- `bira feature note <id> <message>`
- `bira feature delete <id>`

---

## Planning Artifact (Ideas, Bugs & Features)

All three entity types support an optional Markdown sidecar:

- **Idea**: early thinking, research notes, rough sketches before promotion.
- **Bug**: reproduction steps, investigation findings, fix plan.
- **Feature**: full implementation plan the agent will execute.

When an idea is promoted to a feature, its sidecar (if present) is copied to the new feature as a starting point.

`show --json` includes a `plan` field with the raw Markdown string for all three types. When no sidecar exists, `plan` is `""`.

### Storage layout

```
~/.bira/
  projects/
    <project-id>/
      meta.json
      ideas/
        <idea-id>.json
        <idea-id>.plan.md       ← optional Markdown sidecar
      bugs/
        <bug-id>.json
        <bug-id>.plan.md        ← optional Markdown sidecar
      features/
        <feature-id>.json
        <feature-id>.plan.md    ← optional Markdown sidecar
```

---

## Project & Context

`bira init` no longer creates a backlog feature. It creates the project directory and `.bira` pointer file.

`bira context [--full]` shows:
- Default: project name, open idea count, open bug count, open feature count
- `--full`: breakdown by status for each entity type

`bira project list/show/delete` remain unchanged.

---

## Cross-Project Bug Filing

The `-p` / `--project` flag on `bira bug create` accepts either a project ID or a **project slug** (name, lowercased and hyphenated):

```sh
bira bug create -p my-other-tool "Parser crashes on empty input" \
  --desc "Repro: run parser with empty string. Stack trace follows." \
  --reported-by "copilot@my-current-tool"
```

Project slug resolution: scan all `meta.json` files in `~/.bira/projects/` and match on normalised name.

---

## CLI Design Invariants (preserved from today)

- `--json` on every command → clean JSON to stdout
- Errors → stderr only
- No interactive prompts
- Mutating commands echo the affected entity on stdout
- Exit codes: `0` success · `1` general error · `2` not found
- IDs: 8-char hex
- Empty collections serialize as `[]` (never `null`)

---

## What is Removed

| Old Entity/Command | Disposition |
|---|---|
| `Feature` (old model) | Replaced by new Feature + Idea split |
| `Task` | Removed entirely |
| `task add/list/show/update/delete/done/claim/release/note` | Removed |
| `bira task --ready` / readiness logic | Removed |
| `bira feature add --backlog` / IsBacklog | Removed |
| `feature note` | Ported to new Feature |
| `session start/end/list/show` | Removed (v1); deferred to future |
| `feature claim/release`, `bug claim/release` | Removed (v1); deferred to future |

---

## Implementation Order

1. **Models** — Idea, Bug, Feature (new); remove Session, Task
2. **Store layer** — CRUD + plan sidecar read/write + slug resolution
3. **CLI commands** — idea, bug, feature, context (update), project (minor update); remove session/task
4. **MCP server** — `bira mcp` subcommand exposing all store operations as MCP tools over stdio
5. **Tests** — integration tests covering all CLI lifecycles + MCP tool smoke tests
6. **SKILL.md** — comprehensive agent guide covering both CLI and MCP interfaces
7. **README** — full rewrite

---

## SKILL.md

The bira skill will live at `skills/bira/SKILL.md` in the repository. It must be thorough enough that any code agent (Copilot, Claude, Codex, etc.) can use bira correctly via either interface without reading the source code.

The SKILL.md will have two parallel sections:

### Interface selection
- **MCP server** — preferred for IDE-embedded agents (Copilot in VS Code, Cursor, Claude Desktop). Provides structured tool calls, no shell required.
- **CLI** — preferred for shell-based agents, scripts, and terminal workflows. Use `--json` for machine-readable output.

Both interfaces read/write the same `~/.bira` store. They can be used interchangeably.

### Trigger conditions
An agent should invoke the bira skill when asked to (or when it makes sense to):
- "File a bug against \<tool\>"
- "Capture this idea in bira"
- "Check what features are ready to work on"
- "Triage this idea/bug/feature"
- "Start implementing a feature"
- "Mark the current bug as fixed"

### Workflow recipes (CLI)

**1. Quick idea capture**
```sh
bira idea add "My idea title" --desc "Optional detail" --tags "tag1,tag2"
```

**2. Refine an idea into a feature proposal**
```sh
echo "## Background\n...\n## Approach\n..." | bira idea plan <id> --stdin
bira idea promote <id> --json
```

**3. Cooperative triage loop (agent proposes, human confirms)**
```sh
bira idea show <id> --json
bira idea triage <id> --priority high --tags "ux,perf" --json
# adjust if human disagrees, then promote:
bira idea promote <id> --json
```

**4. File a cross-project bug**
```sh
bira bug create -p other-tool "Parser crashes on empty input" \
  --desc "Steps to reproduce: ..." \
  --reported-by "copilot@my-tool" --json
```

**5. Triage a bug**
```sh
bira bug triage <id> --criticality high --tags "parser,crash" --json
```

**6. Implement a feature**
```sh
bira feature list --status triaged --json
bira feature show <id> --json | jq -r '.plan'
bira feature start <id> --json
# ... implement ...
bira feature done <id> --json
```

**7. Fix a bug**
```sh
bira bug start <id> --json
# ... fix ...
bira bug done <id> --json
# or: bira bug wont-fix <id> --note "Out of scope" --json
```

### Workflow recipes (MCP)

The MCP server exposes the same operations as structured tool calls. Example sequence for implementing a feature:

```
list_features(status: "triaged")         → pick an item
get_feature_plan(id: "<id>")             → read the plan
start_feature(id: "<id>")               → mark in-progress
set_feature_plan(id: "<id>", plan: "…") → update plan as work progresses
done_feature(id: "<id>")                → mark complete
```

### Triage guide

Triage is a **cooperative loop** — an agent proposes values and the human confirms or adjusts. An agent should:
1. Read the current item (`show` / `get_*`)
2. Analyse the description and any plan sidecar
3. Propose triage values and explain the reasoning to the human
4. Update again if the human requests changes

| Entity | Triage fields | Effect |
|---|---|---|
| Idea | `priority` (low/medium/high), tags | Makes the idea sortable for promotion |
| Bug | `criticality` (low/medium/high/critical), tags | Agents can filter by criticality |
| Feature | `impact` (low/medium/high), `complexity` (low/medium/high), tags | Agents can rank by impact/effort ratio |

### JSON output contract (CLI)
- All commands accept `--json` → clean JSON to stdout
- Errors go to **stderr** only
- Exit codes: `0` success · `1` error · `2` not found
- Empty arrays serialize as `[]`, never `null`
- `show --json` always includes a `plan` field (raw Markdown string, `""` if no sidecar)

### Cross-project filing
- Use `-p <slug>` where slug = project name lowercased and hyphenated (e.g. `my-other-tool`)
- Or `-p <8-char-hex-id>` if the exact ID is known
- List known projects: `bira project list --json`

---

## Open Questions / Future Considerations

- **Session-based claiming** — when multi-agent support becomes necessary, sessions and claim/release mechanics can be added on top of the status-driven model without breaking the v1 API. Deferred by design.
- **Export/import** — a `bira export` command to dump a project as a portable JSON bundle. Fits naturally before any git-like remote sync.
- **Bug ↔ Feature link** — should a bug be linkable to the feature that fixes it? Deferred.
- **Notification hooks** — a post-create hook so an agent in another repo can be notified when a bug is filed against it. Deferred.
- **Remote sync** — git-inspired push/pull to a shared remote. Explicitly out of scope for v1, but the storage layout should not prevent it.
