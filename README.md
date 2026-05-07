![bira Logo](img/bira_logo_small.png)

# bira

AI-agent optimised CLI for capturing and coordinating **ideas**, **bugs**, and **features**. Designed as a lightweight support tool for agentic development workflows â€” bira tracks intent, not execution. Agents manage their own implementation steps.

## Install

**Prerequisites:** [Go](https://go.dev/dl/) 1.26+

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
bira init                                    # create project, auto-name from directory

# Capture an idea quickly
bira idea add "Add dark mode"

# File a bug against another tool
bira bug create -p other-tool "Crashes on empty input" --reported-by "copilot"

# Add a feature directly
bira feature add "OAuth2 login" --desc "Support GitHub and Google providers"

# Check project state
bira context --full
```

**Agent workflow:**

```sh
# Find triaged work
bira feature list --status triaged --json

# Read the plan, start implementing
bira feature show <id> --json | jq -r '.plan'
bira feature start <id> --json

# Write/update plan as work progresses
echo "## Plan\n..." | bira feature plan <id> --stdin

# Mark done
bira feature done <id> --json
```

---

## Agent interfaces

bira exposes two complementary interfaces that share the same `~/.bira` store:

### CLI
The `bira` binary. All commands accept `--json` for machine-readable output. Use with shell-based agents, scripts, and terminal workflows.

### MCP server
`bira mcp` starts a [Model Context Protocol](https://modelcontextprotocol.io) server over **stdio**. The IDE extension spawns it as a child process â€” no port, no daemon.

**Config** (`mcp.json` or VS Code settings):
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

See [`skills/bira/SKILL.md`](skills/bira/SKILL.md) for the full agent guide covering both interfaces.

---

## Commands

All commands accept `--json` to emit machine-readable JSON and `--project <id>` to override the project resolved from `.bira`.

### Global flags

| Flag | Description |
|------|-------------|
| `--json` | Output JSON instead of human-readable text |
| `--project <id>` | Override project context (default: reads `.bira` walking up from cwd) |
| `--version` | Print version and exit |

---

### `bira init`

```sh
bira init [--name <name>]
```

Creates `~/.bira/projects/<id>/meta.json` and writes a `.bira` pointer file in the current directory. Auto-detects project name from the directory name.

---

### `bira idea`

```sh
bira idea add <title> [--desc <text>] [--tags t1,t2]
bira idea list [--status inbox|triaged|promoted|rejected]
bira idea show <id>
bira idea triage <id> --priority low|medium|high [--tags t1,t2]
bira idea promote <id>           # converts to feature; plan sidecar carried over
bira idea plan <id>              # print plan sidecar
bira idea plan <id> --stdin      # write plan from stdin
bira idea note <id> <message>
bira idea delete <id>
```

**Lifecycle:** `inbox â†’ triaged â†’ promoted / rejected`

---

### `bira bug`

```sh
bira bug create [-p <id|slug>] <title> [--desc <text>] [--reported-by <label>] [--tags t1,t2]
bira bug list [-p <id>] [--status <status>] [--criticality <level>]
bira bug show <id>
bira bug triage <id> --criticality low|medium|high|critical [--tags t1,t2]
bira bug start <id>              # status â†’ in-progress
bira bug done <id>               # status â†’ fixed
bira bug wont-fix <id> [--note <text>]
bira bug plan <id>               # print plan sidecar
bira bug plan <id> --stdin       # write plan from stdin
bira bug note <id> <message>
bira bug delete <id>
```

**Lifecycle:** `open â†’ triaged â†’ in-progress â†’ fixed / wont-fix`

`-p` on `create` accepts a project **name slug** (lowercase, hyphenated) or exact project ID â€” enables cross-repo bug filing without leaving the current context.

---

### `bira feature`

```sh
bira feature add <title> [--desc <text>] [--tags t1,t2]
bira feature list [--status <status>]
bira feature show <id>
bira feature triage <id> --impact low|medium|high --complexity low|medium|high [--tags t1,t2]
bira feature start <id>          # status â†’ in-progress
bira feature done <id>           # status â†’ done
bira feature reject <id> [--note <text>]
bira feature plan <id>           # print plan sidecar
bira feature plan <id> --stdin   # write plan from stdin
bira feature note <id> <message>
bira feature delete <id>
```

**Lifecycle:** `proposed â†’ triaged â†’ in-progress â†’ done / rejected`

---

### `bira context`

```sh
bira context [--full]
```

Default: project name and counts of ideas, bugs, and features.  
`--full`: breakdown by status for each entity type.

---

### `bira project`

```sh
bira project list
bira project show [<id>]
bira project delete <id> --yes
```

---

### `bira mcp`

```sh
bira mcp
```

Starts the MCP stdio server. 26 tools covering all CRUD and lifecycle operations on ideas, bugs, and features. See [`skills/bira/SKILL.md`](skills/bira/SKILL.md) for the full tool list.

---

## Plan sidecars

Every idea, bug, and feature has an optional Markdown sidecar (`<id>.plan.md`) stored alongside its JSON data file.

- **Idea plan** â€” early thinking, research notes
- **Bug plan** â€” reproduction steps, investigation findings, fix plan
- **Feature plan** â€” implementation plan the agent executes

`show --json` always includes a `plan` field (raw Markdown string, `""` if absent). When an idea is promoted to a feature, its plan sidecar is automatically copied to the new feature.

---

## Data storage

All data lives in `~/.bira/` â€” no database, no daemon. Set `BIRA_HOME` to override.

```
~/.bira/
  projects/
    <project-id>/
      meta.json
      ideas/
        <idea-id>.json
        <idea-id>.plan.md     â† optional
      bugs/
        <bug-id>.json
        <bug-id>.plan.md      â† optional
      features/
        <feature-id>.json
        <feature-id>.plan.md  â† optional
```

`.bira` at your repo root holds the `project_id` â€” a lightweight pointer into `~/.bira`.

---

## AI agent design invariants

- `--json` on **every** command â†’ clean JSON to stdout
- All errors go to **stderr** â€” stdout is always pure data
- **No interactive prompts** â€” all input is via flags
- Mutating commands echo the affected entity on stdout
- Exit codes: `0` success Â· `1` error Â· `2` not found
- IDs are 8-character hex strings
- **Empty collections always serialize as `[]`** â€” never `null`

---

## Development

Prerequisites:

- Go 1.26+
- Task v3, installed from the official instructions: <https://taskfile.dev/docs/installation>
- Docker or Podman for `task smoke` and `task ci`

```sh
task test     # native Go tests
task build    # build local binary into dist/
task install  # copy dist/bira to ~/.bira/bin
task smoke    # run Smoko specs; .smokorc builds the test image
task ci       # run test, build, and smoke
task cross    # build the full OS/architecture matrix into dist/
task clean    # remove dist/
```

Equivalent native Go test command for debugging Task itself:

```sh
cd src && go test ./... -v -count=1
```

Tests use isolated environments via `BIRA_HOME` - each test gets its own temp directory. Test isolation is enforced via `NewRootCmd()` (fresh command tree per test, no shared state).
### Bump version

Edit [VERSION](VERSION) (single line, semver). The next build stamps `bira --version` as `<version>+<git-short-hash>`.

| Build method | `bira --version` |
|---|---|
| `task build` / install scripts | `0.1.0+abc1234` |
| Plain `go build` (no ldflags) | `dev+none` |
