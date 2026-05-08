![bira Logo](img/bira_logo_small.png)

# bira

bira is a CLI and MCP server for capturing ideas, bugs, and feature work in
agent-oriented development workflows.

Use bira when you want a small local record of intent while humans or agents do
the actual implementation work elsewhere. It stores data in plain files under
`~/.bira` by default and can emit JSON for automation.

## Install

Prerequisites:

- Go 1.26+
- Git

Linux / macOS:

```sh
git clone https://github.com/nilsgs/bira.git
cd bira
./install.sh
```

Windows PowerShell:

```powershell
git clone https://github.com/nilsgs/bira.git
cd bira
.\install.ps1
```

The installer builds from source, copies `bira` to `~/.bira/bin`, and updates
your user `PATH` where supported.

## Quick Start

```sh
bira init
bira idea add "Add dark mode"
bira bug create "Crash on empty input" --reported-by "copilot"
bira feature add "OAuth2 login" --desc "Support GitHub and Google providers"
bira context --full
```

## Usage

Use `--help` for the full command surface:

```sh
bira --help
bira idea --help
bira bug --help
bira feature --help
```

Common workflows:

```sh
bira idea list
bira idea promote <id>
bira feature triage <id> --impact high --complexity medium
bira feature start <id>
bira feature done <id>
bira context --json
```

All commands accept `--json` for machine-readable output. `--project <id>` can
be used to target a project explicitly instead of resolving it from the current
directory.

`bira mcp` starts the Model Context Protocol server over stdio for MCP-aware
tools.

By default, bira stores data under `~/.bira`. Set `BIRA_HOME` to use another
location.

## Docs

- [Expanded usage](docs/usage.md)
- [Agent and MCP guide](skills/bira/SKILL.md)

## Development

Prerequisites:

- Go 1.26+
- Task v3: <https://taskfile.dev/docs/installation>
- Docker or Podman for `task smoke` and `task ci`

Common tasks:

```sh
task test     # run native Go tests
task build    # build the local binary into dist/
task install  # build and copy the binary to the user install directory
task smoke    # run Smoko specs
task ci       # run test, build, and smoke
task cross    # build the full OS/architecture matrix into dist/
task clean    # remove dist/
```

Smoke specs use tags for focused runs, for example `smoko run specs/ --tag json`.

The version is read from `VERSION` and stamped into the binary at build time.

## License

MIT. See [LICENSE](LICENSE).
