# Findings: 0e42b962 Add smoko specs for all commands

Date: 2026-04-07
Feature ID: 0e42b962
Feature Branch: `feature/0e42b962-smoko-specs`

---

## Bugs

*(No bira CLI bugs encountered.)*

---

## Pain Points / Gaps

- **`bira feature note` / `bira task note` output is not machine-readable**
  - Symptom: `bira feature note <id> "text"` prints a human-readable confirmation but no `--json` flag is available, so there is no structured output to assert on in specs.
  - Impact: Note specs can only assert on the human-readable string (e.g. `"Note added"`), not the stored note content. Verifying the note was persisted requires a separate `bira feature show --json` call.
  - Workaround: Two-step check — add note, then `show --json` and assert on `notes` array.

- **No `bira task list --feature backlog` shorthand**
  - Symptom: To list tasks not assigned to any feature you must use `bira task list` without `--feature`, which returns all tasks. There is no way to filter to only backlog (unassigned) tasks.
  - Impact: Spec for "task in backlog" is slightly imprecise — it asserts the task appears in `bira task list` rather than a dedicated backlog view.
  - Workaround: Use `bira task list` and check that the task name appears.

---

## Suggestions

- **Add `--json` to `bira feature note` and `bira task note`**
  - Solves: "note output is not machine-readable" pain point above.
  - Proposed behaviour: `bira feature note <id> "text" --json` returns the updated feature object (or at minimum `{"id": "...", "note": "text"}`), matching the pattern of other mutating commands.
  - Example: `bira feature note abc123 "found a bug" --json`

- **Add `bira task list --feature ""` or `--no-feature` to filter backlog tasks**
  - Solves: "no backlog filter" pain point above.
  - Proposed behaviour: `bira task list --no-feature` (or `--feature ""`) returns only tasks not assigned to any feature, mirroring how `--feature <id>` filters to a specific feature.
  - Example: `bira task list --no-feature --json`

---

## What Worked Well

- **`--json` flag on most commands**: Nearly every read and write command supports `--json`, making ID extraction and structured assertions straightforward in specs. The consistent pattern of `grep -o '"id": "[^"]*"' | head -1 | sed 's/"id": "\(.*\)"/\1/'` worked reliably without jq.

- **`bira task claim` readiness enforcement**: The CLI correctly blocks claiming tasks that lack `desc`, `assign`, or `criteria`. This made the "claim on non-ready task fails" scenario easy to write and trustworthy.

- **`bira feature delete --move-tasks-to`**: Error message clearly names the required flag when tasks exist. Testable and informative.

---

## Commit References

- `989b3f4` — d4f98db6…b66f2a6a Add smoko specs for all commands (66 scenarios, 66 passing)
