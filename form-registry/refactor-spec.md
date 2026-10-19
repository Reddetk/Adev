# REFACTOR_SPEC — Refactor Spec Form

| field | value |
| --- | --- |
| formType | `REFACTOR_SPEC` |
| stage | `08.R` |
| owner | `dev` |
| route | `greenfield`, `change` |
| mode | `full` |
| inputs | `VERDICT` (fail), `TEST_CONTRACT`, object registry |
| next | `VERDICT` |

## Template

### Paragraph 1 — Failed run

**Scope:** The verdict that starts the refactor.

**Questions:**
1. Which verdict failed?
2. Which task does the verdict belong to?
3. Which implementation produced the failure?

**Answer:** `payload.verdictRef` — `Ref`.
**Answer:** `payload.taskRef` — `task.id`.
**Answer:** `payload.implementationRef` — `Ref`.

### Paragraph 2 — Violation set

**Scope:** The violations that the refactor must close.

**Questions:**
1. Which violation does the refactor close?
2. Which test shows the violation?
3. Which severity applies?

**Answer:** `payload.violations[]` — `{ violationId, testId, severity }`.

### Paragraph 3 — Allowed files

**Scope:** The target-system objects that the refactor may change.

**Questions:**
1. Which `objectId` may the refactor change?
2. Which test object is forbidden?

**Answer:** `payload.allowedFiles[]` — `TargetRef[]` without `kind: test`.

### Paragraph 4 — Fix steps

**Scope:** One ordered action for one target.

**Questions:**
1. Which step number applies?
2. Which target does the step change?
3. Which action does the step do?
4. Which violation does the step close?

**Answer:** `payload.steps[]` — `{ order, targetObjectId, action, text: Text, closesViolationId }`.

### Paragraph 5 — Preservation rule

**Scope:** The tests that must not change.

**Questions:**
1. Must the test set stay unchanged?
2. Which test IDs must pass after the refactor?

**Answer:** `payload.preserveTests` — always `true`.
**Answer:** `payload.mustPass[]` — `test.id` values.

## Gates

1. `preserveTests` is `true`. A `false` value rejects the form.
2. `allowedFiles` contains no `kind: test`.
3. Each `verdictRef.verdict` is `fail`.
4. Each violation from the verdict has a closing step.
5. Each step names one `targetObjectId` and one action.
6. `mustPass` equals the test set of the failed verdict.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `module` | change | Register the change in `changedBy`. |
| `config` | change | Register the change in `changedBy`. |
| `pipeline` | change | Register the change in `changedBy`. |

## Specificity note

A refactor that changes a test is a new requirement, not a refactor. The form rejects it.
