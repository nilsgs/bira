# Agent Workflow for bira Development

This document codifies the workflow for agents developing bira features.

## Pre-Implementation Gate

**Do NOT write any code until all four steps below are complete.** Skipping this gate means
work is untracked and untraceable — do not proceed past this section without confirming each item.

- [ ] `bira feature add` — feature created, ID captured
- [ ] `git checkout -b feature/<id>-<name>` — branch created and checked out
- [ ] `bira task add` × N — every planned task registered in bira, IDs captured
- [ ] `findings/<id>-<name>.md` created — see the **findings skill** for template and guidance

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

Also create the findings file now (see the **findings skill** for the template):

```bash
# touch findings/<feature-id>-<short-name>.md
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

4a. **Update smoko specs**

   If the task added, changed, or removed CLI commands or flags, update the relevant spec file in `specs/`:

   - `specs/init.smoko` — `bira init`
   - `specs/context.smoko` — `bira context`
   - `specs/project.smoko` — `bira project`
   - `specs/feature.smoko` — `bira feature`
   - `specs/task.smoko` — `bira task`
   - `specs/session.smoko` — `bira session`

   Add new scenarios for new behaviours. Update or remove scenarios for changed/removed behaviours.
   See the **smoko skill** for the spec DSL.

> **STOP after each task.** Complete steps 1–5 fully for one task before starting the next.
> One task = one commit + one `bira task done`. Do not batch across tasks.

4. **Commit with bira metadata**
   ```bash
   git commit -m "feat(f2e30189): <feature name> - <task-id> <task name>

   <task description — one or two sentences explaining what this task does>

   Changes:
   - <brief bullet summary of what was added, changed, or removed>
   - <another change if applicable>

   Completes task: <task-id>
   Feature: <feature-id> <feature-name>
   "
   ```

   Example commit message:
   ```
   feat(f2e30189): auth-system - 3e973703 Add JWT validation middleware

   Validates Bearer tokens on every protected route using the shared JWKS endpoint.
   Rejects expired or malformed tokens with 401 before the handler runs.

   Changes:
   - Add JWTMiddleware to the middleware chain in server.go
   - Add ValidateToken() helper in internal/auth/jwt.go
   - Add unit tests for expired, malformed, and valid token cases

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

Capture bira CLI observations (bugs, pain points, suggestions) as you work in:

```
findings/<feature-id>-<short-name>.md
```

See the **findings skill** for the template and guidance.

### 6. Complete the Feature

Once all tasks are done:

1. **Verify tests and smoko specs are comprehensive and green**

   New functionality must be covered by tests. Run the full suite:
   ```bash
   make test-local
   ```
   Only proceed when all tests pass. If any fail, fix the implementation first.

   Then run the full smoko suite to catch any regressions in CLI behaviour:
   ```bash
   make smoko
   ```
   All scenarios must pass. If any fail, fix the implementation or update the specs before proceeding.

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

## Workflow Summary

```
bira feature add          → feature-id
git checkout -b feature/... → branch
bira task add (multiple)  → task-ids
Loop:
  bira task update --status in-progress
  implement work
  run tests (must be green)
  update specs/  (add/change/remove smoko scenarios to match behaviour)
  git commit -m "feat(<id>): ... - <task-id> <task name>\n\n<description>\n\nChanges:\n- ...\n\nCompletes task: <task-id>\nFeature: <feature-id> <feature-name>"
  bira task done
Record findings in findings/<id>-<name>.md
make test-local           (all unit tests green)
make smoko                (all smoke specs green — catches regressions)
bira feature update --status done
git push
```
