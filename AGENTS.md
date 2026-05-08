# bira Agent Notes

## Layout

- `src/main.go`: CLI entry point.
- `src/cmd/`: Cobra commands for `init`, `idea`, `bug`, `feature`, `context`, `project`, and `mcp`.
- `src/internal/models/`: project, idea, bug, and feature data models.
- `src/internal/store/`: JSON file storage and plan sidecar helpers.
- `src/internal/config/`: `.bira` project pointer handling.
- `specs/`: Smoko smoke specs.
- `skills/bira/`: agent and MCP workflow guidance.

## Build And Test

Use Task targets for normal validation:

```sh
task test
task build
task smoke
task ci
```

Raw Go fallback for focused debugging:

```sh
cd src
go test ./... -v -count=1
```

## Smoke Tests

Smoke specs live under `specs/` and are run with Smoko:

```sh
task smoke
```

`.smokorc` owns the image build. Do not duplicate Docker build commands in Task
or docs unless the project workflow changes.

## Contracts

- Every command accepts `--json`.
- JSON output goes to stdout.
- Errors go to stderr.
- Empty collections serialize as `[]`.
- Exit code `0` means success.
- Exit code `1` means general error.
- Exit code `2` means not found.
- `show --json` includes a `plan` field.

Entity lifecycles:

```text
idea: inbox -> triaged -> promoted / rejected
bug: open -> triaged -> in-progress -> fixed / wont-fix
feature: proposed -> triaged -> in-progress -> done / rejected
```

## Documentation

- Keep `README.md` as the concise user and developer front door.
- Put expanded user workflows in `docs/usage.md`.
- Keep agent and MCP workflow detail in `skills/bira/SKILL.md`.
- Keep Markdown docs ASCII-only unless a non-ASCII character is deliberate.

## Versioning

The version comes from `VERSION` and is stamped into builds by the Task build
scripts.
