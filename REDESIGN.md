# Bira — Redesign Plan

## Problem Statement

The current bira model (Projects → Features → Tasks, with multi-agent session/claim/release) was designed for a world where the AI agent needs a managed todo list. Modern agents (Copilot, Claude, Codex) maintain their own internal task lists during implementation — bira duplicating that is unnecessary friction. The real value bira can offer is:

1. **Lightweight capture** — a quick way to record ideas, bugs, and proposals without context-switching.
2. **Loose coupling between planning and implementation** — bira owns the "what and why", the agent decides "how".
3. **Cross-repo coordination** — filing a bug against another project without leaving the current context.
4. **Triage** — a structured way to prioritise and enrich captured items so agents can confidently pick up the right work.
5. **Claim safety** — preventing two agent sessions from picking up the same item simultaneously.

---

## Role Boundary

**bira is a support tool. It does not fill itself with content.**

The responsibility for creating well-formed, actionable bira records is shared between humans and agents in a collaborative loop:

- **Humans** — capture quick ideas, file rough bug reports, review and approve triage suggestions, decide priorities.
- **Agents** — enrich descriptions, propose triage (priority, criticality, impact/complexity), write plan sidecars, file bugs against other tools, claim and implement features.

The typical flow is **iterative**: a human captures something raw, an agent refines it and proposes triage, the human reviews and adjusts, back and forth until the item is ready. Neither party owns the process end-to-end.

bira's job is to provide a clean, reliable CLI that makes this cooperative loop frictionless. A well-maintained `SKILL.md` (see below) is therefore as important as the CLI itself — it is the interface contract that lets any agent participate in the loop correctly without guessing.

---

## Core Design Principles

- **bira tracks intent, not execution.** No tasks. Agents manage their own implementation steps.
- **Three first-class entity types**: Idea, Bug, Feature — each with its own lifecycle and triage fields.
- **Session-based claiming** remains the coordination primitive (sessions are already the right abstraction; agent identity is non-deterministic).
- **Storage stays local** (`~/.bira`), but the design should not preclude a future git-like sync mechanism.
- **Markdown plan sidecars** on all three entity types — free-form, writable by human or agent.
- **All design invariants from today are preserved**: `--json`, stderr for errors, exit codes, no interactive prompts, empty collections as `[]`.

---

## Entity Lifecycles

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

**Fields**: id, project_id, title, description, criticality (after triage: `low|medium|high|critical`), reported_by (free-text, optional), tags, notes[], plan_path (optional .md sidecar), claimed_by (session_id), claimed_at, status, created_at, updated_at

**Key commands**:
- `bira bug create [-p <id|slug>] <title> [--desc <text>] [--reported-by <label>]`
  - `--desc -` reads description from stdin (multiline support)
- `bira bug list [-p <id>] [--status <status>] [--criticality <level>] [--unclaimed]`
- `bira bug show <id>`
- `bira bug triage <id> --criticality <low|medium|high|critical> [--tags t1,t2]`
- `bira bug claim <id> [--session <id>]`
- `bira bug done <id>` — marks fixed
- `bira bug release <id> [--note <text>] [--status open|wont-fix]`
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

**Fields**: id, project_id, title, description, impact (`low|medium|high`), complexity (`low|medium|high`), tags, notes[], claimed_by (session_id), claimed_at, plan_path (relative path to .md sidecar), status, promoted_from (idea_id, optional), created_at, updated_at

**Key commands**:
- `bira feature add <title> [--desc <text>] [--tags t1,t2]`
- `bira feature list [--status <status>] [--unclaimed]`
- `bira feature show <id>`
- `bira feature triage <id> --impact <low|medium|high> --complexity <low|medium|high> [--tags t1,t2]`
- `bira feature plan <id> [--edit | --stdin]`
  - No flags: prints the current plan (or empty state prompt)
  - `--edit`: opens sidecar in `$EDITOR`
  - `--stdin`: reads plan content from stdin (agent-friendly)
- `bira feature claim <id> [--session <id>]`
- `bira feature done <id>`
- `bira feature release <id> [--note <text>]`
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
      sessions/
        <session-id>.json
```

---

## Sessions

Sessions remain conceptually the same — a stable identity for claiming work. Changes:
- Sessions are no longer scoped to a single project (a session can claim items across projects).
- `session start/end/list/show` are kept.
- Claims exist on Bug and Feature (not on Idea).
- Stale claim timeout remains configurable via `.bira` (`claim_timeout_minutes`, default 1440 / 24h).

---

## Project & Context

`bira init` no longer creates a backlog feature. It creates the project directory and `.bira` pointer file.

`bira context [--full]` shows:
- Default: project name, open idea count, open bug count, open feature count
- `--full`: breakdown by status for each entity type, plus claimable counts

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
- `$BIRA_SESSION` sets default session for claim/release/done

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
| Session scoped to project | Sessions become global |

---

## Implementation Order

1. **Models** — Idea, Bug, Feature (new), Session (update)
2. **Store layer** — CRUD + plan sidecar read/write + slug resolution
3. **Commands** — idea, bug, feature, session (update), context (update), project (minor update)
4. **Tests** — integration tests covering all lifecycles
5. **SKILL.md** — comprehensive agent guide (see below)
6. **README** — full rewrite

---

## SKILL.md

The bira skill will live at `skills/bira/SKILL.md` in the repository. It must be thorough enough that any code agent (Copilot, Claude, Codex, etc.) can use bira correctly without reading the source code.

### Trigger conditions
An agent should invoke the bira skill when asked to (or when it makes sense to):
- "File a bug against \<tool\>"
- "Capture this idea in bira"
- "Check what features are ready to work on"
- "Triage this idea/bug/feature"
- "Claim and implement a feature"
- "Mark the current bug as fixed"

### Workflow recipes

**1. Quick idea capture**
```sh
bira idea add "My idea title" --desc "Optional detail" --tags "tag1,tag2"
```

**2. Refine an idea into a feature proposal**
```sh
# Attach a plan to the idea first
echo "## Background\n...\n## Approach\n..." | bira idea plan <id> --stdin
# Then promote to feature
bira idea promote <id> --json   # returns the new feature record
```

**3. Cooperative triage loop (agent proposes, human confirms)**
```sh
# Agent reads the raw item
bira idea show <id> --json

# Agent proposes triage (explain reasoning to the human)
bira idea triage <id> --priority high --tags "ux,perf" --json

# If human disagrees, agent adjusts:
bira idea triage <id> --priority medium --json

# Once happy, human or agent promotes to feature
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

**6. Claim and implement a feature**
```sh
export BIRA_SESSION=$(bira session start --label "copilot" --json | jq -r '.id')
bira feature list --status triaged --unclaimed --json   # find ready work
bira feature show <id> --json | jq -r '.plan'          # read the plan
bira feature claim <id> --json
# ... implement ...
bira feature done <id> --json
bira session end $BIRA_SESSION --json
```

**7. Release a feature back to the pool**
```sh
bira feature release <id> --note "Blocked by upstream issue" --json
```

### Triage guide

Triage is a **cooperative loop** — an agent proposes values and the human confirms or adjusts. An agent should:
1. Read the current item: `bira <type> show <id> --json`
2. Analyse the description and any plan sidecar
3. Propose triage values by calling the triage command
4. Summarise the reasoning to the human and invite feedback
5. Update again if the human requests changes

| Entity | Fields to set during triage | Effect |
|---|---|---|
| Idea | `priority` (low/medium/high), tags | Makes the idea sortable for promotion |
| Bug | `criticality` (low/medium/high/critical), tags | Agents can filter by criticality |
| Feature | `impact` (low/medium/high), `complexity` (low/medium/high), tags | Agents can rank by impact/effort ratio |

```sh
bira idea triage    <id> --priority medium --tags "ux,perf" --json
bira bug triage     <id> --criticality high --tags "parser,crash" --json
bira feature triage <id> --impact high --complexity medium --tags "auth" --json
```

### JSON output contract
- All commands accept `--json` → clean JSON to stdout
- Errors go to **stderr** only
- Exit codes: `0` success · `1` error · `2` not found
- Empty arrays serialize as `[]`, never `null`
- `show --json` always includes a `plan` field (raw Markdown string, `""` if no sidecar)

### Cross-project filing
- Use `-p <slug>` where slug = project name lowercased and hyphenated (e.g. `my-other-tool`)
- Or `-p <8-char-hex-id>` if the exact ID is known
- List known projects: `bira project list --json`

### Environment variable
```sh
export BIRA_SESSION=<session-id>   # set once; --session flag is then implicit
```

---

## Open Questions / Future Considerations

- **Export/import** — a `bira export` command to dump a project as a portable JSON bundle. Fits naturally before any git-like remote sync.
- **Bug ↔ Feature link** — should a bug be linkable to the feature that fixes it? Deferred.
- **Notification hooks** — a post-create hook so an agent in another repo can be notified when a bug is filed against it. Deferred.
- **Remote sync** — git-inspired push/pull to a shared remote. Explicitly out of scope for v1, but the storage layout should not prevent it.
