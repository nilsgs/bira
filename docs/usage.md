# bira Usage

bira stores local project intent as ideas, bugs, and features. It is designed
for command-line use, scripts, and agents that need stable JSON output.

## Concepts

- Project: a named workspace tracked by a `.bira` pointer file.
- Idea: early input that may be triaged, promoted, or rejected.
- Bug: a defect report that can be triaged, started, fixed, or marked wont-fix.
- Feature: planned work that can be triaged, started, completed, or rejected.
- Plan sidecar: optional Markdown stored next to an idea, bug, or feature.

## Project Setup

Initialize a project from its repository root:

```sh
bira init
```

Use an explicit name when the directory name is not enough:

```sh
bira init --name "My Project"
```

The command creates a project under `~/.bira/projects/` and writes a `.bira`
file in the current directory. Commands resolve the active project by walking up
from the current directory until they find `.bira`.

## Ideas

```sh
bira idea add "Investigate dark mode"
bira idea list
bira idea show <id>
bira idea triage <id> --priority high
bira idea promote <id>
bira idea reject <id> --note "Out of scope"
```

Idea lifecycle:

```text
inbox -> triaged -> promoted
                  -> rejected
```

## Bugs

```sh
bira bug create "Crash on empty input"
bira bug list
bira bug show <id>
bira bug triage <id> --criticality high
bira bug start <id>
bira bug done <id>
bira bug wont-fix <id> --note "By design"
```

Bug lifecycle:

```text
open -> triaged -> in-progress -> fixed
                              -> wont-fix
```

Use `-p` or `--project` when filing a bug against another project:

```sh
bira bug create -p other-project "Crashes on empty input"
```

The project value can be an exact project ID or a project name slug.

## Features

```sh
bira feature add "OAuth2 login"
bira feature list
bira feature show <id>
bira feature triage <id> --impact high --complexity medium
bira feature start <id>
bira feature done <id>
bira feature reject <id> --note "Not needed"
```

Feature lifecycle:

```text
proposed -> triaged -> in-progress -> done
                                  -> rejected
```

## Plans

Every idea, bug, and feature can have a Markdown plan sidecar:

```sh
bira feature plan <id>
bira feature plan <id> --stdin
```

When an idea is promoted to a feature, its plan sidecar is copied to the new
feature. `show --json` always includes a `plan` field. If no plan exists, the
field is an empty string.

## JSON Output

All commands accept `--json`:

```sh
bira feature list --status triaged --json
bira feature show <id> --json
bira context --full --json
```

Output contract:

- JSON is written to stdout.
- Errors are written to stderr.
- Empty collections are serialized as `[]`.
- Exit code `0` means success.
- Exit code `1` means general error.
- Exit code `2` means not found.

## MCP Server

Start the MCP stdio server:

```sh
bira mcp
```

Example MCP configuration:

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

See `skills/bira/SKILL.md` for the full agent workflow and MCP tool guide.

## Storage

Default layout:

```text
~/.bira/
  projects/
    <project-id>/
      meta.json
      ideas/
        <idea-id>.json
        <idea-id>.plan.md
      bugs/
        <bug-id>.json
        <bug-id>.plan.md
      features/
        <feature-id>.json
        <feature-id>.plan.md
```

Set `BIRA_HOME` to use another storage root.
