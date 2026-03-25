# Agent Workflow for bira Development

This document codifies the workflow for agents developing bira features.

## Pre-Implementation Gate

**Do NOT write any code until all three steps below are complete.** Skipping this gate means
work is untracked and untraceable — do not proceed past this section without confirming each item.

- [ ] `bira feature add` — feature created, ID captured
- [ ] `git checkout -b feature/<id>-<name>` — branch created and checked out
- [ ] `bira task add` × N — every planned task registered in bira, IDs captured
- [ ] `findings/<id>-<name>.md` created from the template (can be mostly empty — fill as you go)

If any item is incomplete, do it now before touching source files.

---

## Prerequisites

- bira CLI is pre-installed (do not build from source)
- Working Git repository
- bira project initialized (`bira init`)

## Feature Implementation Workflow

### 1. Plan the Feature in bira

```bash
bira feature add "<feature name>" --desc "<description>" --json
```

Capture the feature ID (8-char hex) from the JSON response.

### 2. Create a Feature Branch

Branch naming convention: `feature/<feature-id>-<short-name>`

Example: `feature/f2e30189-auth-system`

```bash
git checkout -b feature/f2e30189-auth-system
```

### 3. Create Tasks in bira

Break down the feature into actionable tasks:

```bash
bira task add "<task title>" --feature <feature-id> --json
bira task add "<another task>" --feature <feature-id> --json
```

Capture task IDs as you work.

Also create the findings file now:

```bash
# Create findings/<feature-id>-<short-name>.md from the template at the bottom of this document
```

### 4. Implement and Commit

For each completed task:

1. **Mark task in-progress**
   ```bash
   bira task update <task-id> --status in-progress
   ```

2. **Implement the work**

3. **Test the implementation**
   
   All code changes must be validated by tests before marking a task complete. This ensures:
   - Changes work as intended
   - Regressions are caught early
   - Future maintenance is safer

   Run the relevant test suite:
   ```bash
   # Full test suite
   make test-local
   
   # Or specific package tests
   cd src && go test ./cmd -v
   cd src && go test ./internal/store -v
   ```

   **Green tests are mandatory** — only commit code that passes all tests. If tests fail, update the implementation until all tests pass.

> **STOP after each task.** Complete steps 1–5 fully for one task before starting the next.
> One task = one commit + one `bira task done`. Do not batch across tasks.

4. **Commit with bira metadata**
   ```bash
   git commit -m "feat(f2e30189): <feature name> - <task-id> <task name>

   Completes task: <task-id>
   Feature: <feature-id> <feature-name>
   "
   ```

   Example commit message:
   ```
   feat(f2e30189): auth-system - 3e973703 Add JWT validation middleware

   Completes task: 3e973703
   Feature: f2e30189 auth-system
   ```

5. **Mark task done in bira**
   ```bash
   bira task done <task-id>
   ```

6. **View progress**
   ```bash
   bira context --full --json
   ```

### 5. Document Findings

As you work, capture findings in:

```
findings/<feature-id>-<short-name>.md
```

Example path: `findings/f2e30189-auth-system.md`

See [Findings Format](#findings-format) below.

### 6. Complete the Feature

Once all tasks are done:

1. **Verify tests are comprehensive and green**

   New functionality must be covered by tests. Run the full suite:
   ```bash
   make test-local
   ```
   Only proceed when all tests pass. If any fail, fix the implementation first.

2. **Update README.md**

   Document any new CLI flags, subcommands, or changed behaviours introduced by the feature. Keep the command reference tables and examples in sync with the implementation.

3. **Update the bira skill**

   If the feature added or changed CLI commands, flags, or data model fields, update `.agents/skills/bira/SKILL.md` to reflect the current behaviour. Agents depend on this skill to operate bira correctly.

4. **Ask the user: bump version and publish?**

   Before committing, ask:

   > "Do you want to bump the version and publish a new release?"

   If **yes**:
   1. Read the current version: `cat VERSION`
   2. Determine the new version with the user (patch / minor / major bump per SemVer)
   3. Write the new version: `echo "<new-version>" > VERSION`
   4. Commit the version bump:
      ```bash
      git add VERSION
      git commit -m "chore: bump version to <new-version>"
      ```
   5. Ask the user: "Do you want to run an install script to rebuild and install the binary locally?"
      - **Linux / macOS:** `./install.sh`
      - **Windows:** `.\install.ps1`

   If **no**, skip and proceed to the next step.

5. **Mark the feature done and push**
   ```bash
   bira feature update <feature-id> --status done
   git push origin feature/<feature-id>-<short-name>
   ```

---

## Findings Format

Create a structured findings document for each feature. This accumulates observations about bira's usability, pain points, and opportunities.

```markdown
# Findings: <feature-id> <feature-name>

Date: YYYY-MM-DD
Feature ID: <id>
Feature Branch: `feature/<id>-<short-name>`

---

## What Worked Well

- **Category / Pattern**: Observation about what was intuitive or effective
  - Reasoning or impact
  - Example if applicable

- **Another positive**: Details

---

## Pain Points / Gaps

- **Issue**: Description of what was difficult or missing
  - Symptom: How did this manifest?
  - Impact: What did it prevent or slow down?
  - Workaround: (if applicable) How did you work around it?

- **Another issue**: Details

---

## Neutral Observations

- **Behavior**: Factual observation that's neither positive nor negative
  - Context where this matters

---

## Suggestions

- **Improvement**: Concrete suggestion for enhancing bira
  - Rationale: Why would this help?
  - Example usage: How would the improved workflow work?

---

## Commit References

List commits completed during this feature:

- `<commit-hash>` - <task-id> <task-name>
- `<commit-hash>` - <task-id> <task-name>
```

### Finding Categories

Organize findings one of four ways:

| Category | Purpose |
|----------|---------|
| **What Worked Well** | Positive patterns to preserve; design decisions that enable users |
| **Pain Points / Gaps** | Blockers, missing features, unintuitive UX; these drive improvements |
| **Neutral Observations** | Behavioral facts that don't feel positive/negative but matter in context |
| **Suggestions** | Actionable ideas for bira enhancement (already identified pain → proposed solution) |

### Pain Point Template

Every pain point should include:

1. **Issue**: One-line summary
2. **Symptom**: How did the problem appear?
3. **Impact**: What work did it block or slow?
4. **Workaround**: (if applicable) How to get unstuck

### Example Finding Entry

```markdown
## Pain Points / Gaps

- **No way to bulk-update task status**
  - Symptom: After completing a feature, marking all 7 tasks as `done` requires 7 separate CLI calls
  - Impact: Finishing features is tedious; slows down feature completion tracking
  - Workaround: Use `bira task list --feature <id> --json | jq '.[] | .id'` to get IDs, then loop over them
```

---

## Workflow Summary

```
bira feature add          → feature-id
git checkout -b feature/... → branch
bira task add (multiple)  → task-ids
Loop:
  bira task update --status in-progress
  implement work
  run tests (must be green)
  git commit -m "feat(<id>): ..."
  bira task done
Record findings in findings/<id>-<name>.md
bira feature update --status done
git push
```
