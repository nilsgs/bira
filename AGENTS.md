# bira â€” Agent Bootstrap

**bira** is a CLI and MCP server for capturing ideas, bugs, and feature proposals in agentic workflows.
Agents drive bira. bira tracks intent, not execution steps.

---

## Repo layout

```
src/
  main.go
  cmd/          # cobra commands: init, idea, bug, feature, context, project, mcp
  internal/
    models/     # Idea, Bug, Feature, Project structs
    store/      # JSON file store, plan sidecar helpers, slug resolution
    config/     # .bira pointer file (project_id)
specs/          # Smoko smoke tests (.smoko files)
skills/bira/    # SKILL.md â€” full agent workflow guide
```

---

## Build & test commands

```sh
task build                            # compile binary into dist/
task test                             # native Go unit tests
task smoke                            # build Docker image via .smokorc + run smoke tests
task ci                               # test, build, smoke
cd src && go build ./...              # quick local compile check
cd src && go test ./... -v -count=1   # raw test fallback
```

---

## Data layout

```
~/.bira/
  projects/
    <project-id>/
      meta.json           # { id, name }
      ideas/<id>.json     # Idea entity
      ideas/<id>.plan.md  # optional plan sidecar (plain Markdown)
      bugs/<id>.json
      bugs/<id>.plan.md
      features/<id>.json
      features/<id>.plan.md
```

Project pointer: `.bira` file in the working directory contains `project_id`.
`bira init --name <name>` creates the project and writes the pointer.

---

## MCP server

`bira mcp` starts an MCP stdio server with 21 tools:

| Group | Tools |
|---|---|
| context | `bira_context` |
| ideas | `bira_idea_add`, `bira_idea_list`, `bira_idea_show`, `bira_idea_triage`, `bira_idea_promote`, `bira_idea_reject` |
| bugs | `bira_bug_create`, `bira_bug_list`, `bira_bug_show`, `bira_bug_triage`, `bira_bug_start`, `bira_bug_done`, `bira_bug_wont_fix` |
| features | `bira_feature_add`, `bira_feature_list`, `bira_feature_show`, `bira_feature_triage`, `bira_feature_start`, `bira_feature_done`, `bira_feature_reject` |

All tools mirror the CLI commands. Use MCP tools when in an MCP-aware environment; use the CLI otherwise.
See `skills/bira/SKILL.md` for full tool signatures and workflow recipes.

---

## CLI output contract

- Every command accepts `--json` â†’ structured JSON to stdout
- Errors â†’ stderr only; stdout stays clean
- Exit codes: `0` success Â· `1` error Â· `2` not found
- Empty collections serialize as `[]`, never `null`
- `show --json` always includes a `plan` field (raw Markdown string, `""` if no sidecar)

---

## Entity lifecycles

| Entity | Statuses |
|---|---|
| Idea | `inbox` â†’ `triaged` â†’ `promoted` / `rejected` |
| Bug | `open` â†’ `triaged` â†’ `in-progress` â†’ `fixed` / `wont-fix` |
| Feature | `proposed` â†’ `triaged` â†’ `in-progress` â†’ `done` / `rejected` |

Key JSON fields: `id`, `title` (not `name`), `status`, `plan` (sidecar content on show).

Cross-project bug filing: `bira bug create -p <slug-or-id> "title"` where slug = project name lowercased and hyphenated.

---

## Mandatory workflow rules

1. **Run smoke tests** (`task smoke`) before marking any implementation complete.
2. **Update documentation** whenever CLI commands, entity contracts, or workflows change:
   - `README.md` â€” user-facing command reference
   - `AGENTS.md` â€” this file
   - `skills/bira/SKILL.md` â€” agent workflow guide and JSON contract
