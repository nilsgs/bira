---
name: findings
description: "Use when: writing findings documents, documenting bira CLI experience, recording pain points with bira, logging bira bugs, capturing bira improvement ideas, filling out findings/<id>-<name>.md, evaluating bira usability during development. Guides agents on how to write findings documents that evaluate the bira CLI tool — not summarize the implementation."
---

# findings

A **findings document** records your experience *using the bira CLI tool* during a feature's
development. It is a quality signal — an honest evaluation of bira's usability, correctness,
and developer experience (DX).

**What findings are NOT:**
- A summary of what you built
- A narrative of implementation decisions
- A changelog or commit log

**What findings ARE:**
- Bugs you hit in bira itself
- Friction you felt using bira commands
- Missing features that forced workarounds
- Behaviours that were confusing or unintuitive
- Ideas for making bira better

> **Bias toward critical findings.** Positive feedback is nice but rarely actionable.
> Bugs and pain points drive real improvements. If you only have positive things to say,
> look harder — frictionless sessions are rare.

---

## When to Capture Findings

Capture findings **in the moment**, not at the end of the feature. When something feels wrong,
stop and write it down immediately. Retrospective findings lose detail.

Two fast capture mechanisms during work:

```bash
# Quick note on the current feature (use for anything bira-related)
bira feature note <feature-id> "bira task done failed with exit 1 — task was already in-progress"

# Or write directly to the findings file as you work
```

Transfer rough notes to the structured findings file before pushing.

---

## File Location and Naming

```
findings/<feature-id>-<short-name>.md
```

Example: `findings/f2e30189-auth-system.md`

Create the file at the start of the feature (even if empty). Fill it as you work.

---

## Findings Document Template

```markdown
# Findings: <feature-id> <feature-name>

Date: YYYY-MM-DD
Feature ID: <id>
Feature Branch: `feature/<id>-<short-name>`

---

## Bugs

Actual defects in the bira CLI — wrong output, crashes, incorrect data, exit code mismatches.

- **[Bug title]**
  - Command: `bira <exact command that triggered it>`
  - Expected: What should have happened
  - Actual: What actually happened
  - Reproducible: Yes / No / Sometimes
  - Workaround: (if any) How to get unstuck

---

## Pain Points / Gaps

Friction, missing features, and unintuitive UX — things that slowed you down or required workarounds.

- **[Issue title]**
  - Symptom: How did this manifest while working?
  - Impact: What did it block or slow?
  - Workaround: (if any) How did you get past it?

---

## Suggestions

Concrete improvement ideas — each tied to a bug or pain point above.

- **[Suggestion title]**
  - Solves: Which bug or pain point does this address?
  - Proposed behaviour: How would the improved bira behave?
  - Example: `bira <example of the improved command or output>`

---

## What Worked Well

*(Secondary — only include if something is genuinely worth preserving or highlighting.)*

- **[Pattern or behaviour]**: One sentence on why it was effective.

---

## Commit References

- `<commit-hash>` — <task-id> <task-name>
```

---

## Category Reference

| Category | Priority | What belongs here |
|----------|----------|-------------------|
| **Bugs** | 🔴 Highest | Wrong output, crashes, bad exit codes, data corruption, any bira defect |
| **Pain Points / Gaps** | 🟠 High | Friction, missing commands, clunky UX, forced workarounds |
| **Suggestions** | 🟡 Medium | Improvement ideas grounded in the above |
| **What Worked Well** | 🟢 Low | Positive patterns — only if genuinely noteworthy |

---

## Bug Entry Template

Every bug entry must include:

1. **Command**: The exact `bira` command that triggered the issue
2. **Expected**: What the correct behaviour should be
3. **Actual**: What bira actually did
4. **Reproducible**: Can you trigger it again?
5. **Workaround**: (if applicable) How to proceed despite the bug

### Example Bug Entry

```markdown
## Bugs

- **`bira task done` exits 1 when task is already `done`**
  - Command: `bira task done a3f1c9b2` (task was already marked done)
  - Expected: Idempotent — exit 0, task stays done
  - Actual: Exit code 1 with "task already done" on stderr; breaks scripted loops
  - Reproducible: Yes — reliably triggered by marking done twice
  - Workaround: Check `bira task show <id> --json | jq '.status'` before calling `done`
```

---

## Pain Point Entry Template

Every pain point should include:

1. **Issue**: One-line summary
2. **Symptom**: How did the problem appear during your work?
3. **Impact**: What did it block or slow down?
4. **Workaround**: (if applicable) How to get unstuck

### Example Pain Point Entry

```markdown
## Pain Points / Gaps

- **No bulk status update for tasks**
  - Symptom: Closing out a feature required 7 separate `bira task done` calls
  - Impact: Tedious end-of-feature cleanup; error-prone if one is missed
  - Workaround: `bira task list --feature <id> --json | jq -r '.[].id' | ForEach-Object { bira task done $_ }`
```

---

## Anti-Patterns

Avoid these common mistakes when writing findings:

| Anti-pattern | Why it's wrong | Instead |
|---|---|---|
| "Implemented the login handler" | That's implementation, not bira evaluation | Describe a bira interaction that happened *while* implementing |
| "Everything went smoothly" | Rarely true — and not useful | Look for any friction, even minor; document it |
| Writing findings after merging | Detail is lost; memory is unreliable | Capture in the moment or via `bira feature note` |
| Vague pain point: "bira was slow" | Not actionable | Specify the command, the delay, and the context |
| Suggestions without grounding | Unmoored ideas are low-value | Every suggestion should reference a specific bug or pain point |
