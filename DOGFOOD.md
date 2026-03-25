# Bira Dogfood Notes

Observations collected while using bira to plan, track, and implement its own test/validation harness feature.

---

## What Works Well

- **`bira init --json`** already works via the global `--json` persistent flag — no extra work needed. Good design decision to make it persistent on root.
- **Consistent RunE signatures** — every command handler already receives `cmd *cobra.Command`, making the output threading purely mechanical.
- **`SilenceUsage: true` and `SilenceErrors: true`** on rootCmd — prevents Cobra from printing usage on errors, which keeps stdout clean for JSON consumers.
- **`bira context --full --json`** is well-designed for AI agent consumption — per-feature breakdown makes it easy to understand open work at a glance without needing to list tasks separately.
- **8-char hex IDs** are readable in prompts and logs without being unwieldy.
- **`--project <id>` flag** provides a clean escape hatch from CWD dependency, which is essential for scripting and testing.
- **Separation of stdout (data) / stderr (errors)** is excellent — `--json` output is always clean and pipeable.
- **Backlog feature** is a thoughtful default — ungroomed tasks have a natural home without requiring explicit feature creation.

## Code Quality Observations

- **Code duplication**: `config.ResolveProjectID()` and `cmd/project.go:resolveProjectIDFromFlag()` do nearly identical work. Not blocking for test harness, but worth noting for future cleanup.

---

## Planning Phase Observations

### Pain points / gaps discovered

- **No way to initialize bira without a terminal** — planning the implementation required knowing bira is not yet initialized (`no .bira` found). There's no `bira status` or similar command to check the overall state without already being initialized.
- **`bira context` fails before init** — useful to surface this earlier with a friendlier "not initialized" message that hints at `bira init`.
- **No way to list all tasks across features without feature ID** — `bira task list` without `--feature` doesn't aggregate; you'd need to iterate over feature IDs to get a full task list.
- **`DependsOn` is informational only** — stored but not enforced or surfaced in `context`. For agent use, knowing blocked tasks require attention is valuable.
- **Feature status is manual** — no signal when all tasks under a feature are `done` but the feature itself is still `in-progress`. Easy to forget to close out the feature.
- **No search** — when a project has many tasks, finding a specific one requires listing all and scanning visually. `bira task list --title "some keyword"` would help.

---

## Implementation Phase Observations

### Duplicate task creation via multi-line terminal commands (CRITICAL)
When an AI agent sends multiple `bira task add` commands in a single multi-line terminal block to PowerShell, the shell interprets them inconsistently — some lines get parsed as part of previous commands, some get executed multiple times, and some get swallowed. This created **21 tasks instead of 7** (3x duplication).

**Root cause**: Not a bira bug — it's the interaction between AI terminal tools and PowerShell's multi-line handling. Multi-line `;`-separated commands also fail because shell continuation characters in `--desc` strings break parsing.

**Mitigation for AI agents using bira**: Always send **one `bira` command per terminal invocation**. Never batch `bira` commands in multi-line blocks.

**Potential bira improvement**: An idempotency mechanism (e.g., `--idempotency-key <key>`) or batch import (`bira task import tasks.json`) would eliminate this class of problem entirely.

### `exitNotFound` calls `os.Exit(2)` — untestable
During integration test development, commands that use `exitNotFound()` (task show, task delete on missing IDs) cannot be tested for error cases because they call `os.Exit(2)` directly, which kills the test process. Had to work around this by verifying via list instead.

**Potential bira improvement**: Return errors instead of calling `os.Exit()` — let `Execute()` in root.go handle exit codes. This would make all error paths testable.

### Package-level flag vars leak between test runs
All Cobra flag vars (`taskAddFeature`, `featureListStatus`, etc.) are package-level. Tests must manually reset every flag before each command execution to avoid state leakage. This is fragile — adding a new flag requires updating the test helper.

**Potential bira improvement**: Consider a command factory pattern where each run creates fresh command instances, or at minimum document which vars need resetting.

### `contextFull` flag also leaks
The `contextFull` bool persists between runs. The `--full` flag from one test affects subsequent tests unless explicitly reset. Same root cause as above.

---

## Test Harness Summary

### What was built
- **`BIRA_HOME` env var** — overrides `~/.bira` data directory, enabling per-test isolation via `t.TempDir()` + `t.Setenv()`
- **Writer-threaded output** — all output functions accept `io.Writer` via `cmd.OutOrStdout()`, enabling tests to capture output to buffers
- **28 tests** across 3 packages:
  - `internal/store` — 9 tests (ID generation, JSON save/load, directory operations, BIRA_HOME)
  - `internal/config` — 6 tests (save/load, walk-up discovery, project resolution)
  - `cmd` — 13 integration tests (init, feature CRUD, task CRUD, context, JSON output)
- **Makefile targets** — `make test` (Docker isolation), `make test-local` (native)
