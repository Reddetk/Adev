# TASK_PLAN — Task Plan

| field | value |
| --- | --- |
| formType | `TASK_PLAN` |
| stage | `06` |
| owner | `analyst` |
| route | `greenfield`, `change` |
| mode | `full`, `delta` |
| inputs | `LLD`, `ADR`, `SRS` |
| next | `TEST_CONTRACT` |

## Template

### Paragraph 1 — Task identity

**Scope:** One unit of work.

**Questions:**
1. Which ID identifies the task?
2. Which short title describes the task?
3. Which task type applies?

**Answer:** `payload.tasks[]` — `{ id, title: Text, taskType }`.

### Paragraph 2 — Tree position

**Scope:** The place of the task in the tree.

**Questions:**
1. Which parent task contains this task?
2. Which tasks have no parent?

**Answer:** `payload.tasks[].parentId` — `task.id` or `null`.
**Answer:** `payload.roots[]` — `task.id` values.

### Paragraph 3 — Owner

**Scope:** The single agent that runs the task.

**Questions:**
1. Which agent owns the task?

**Answer:** `payload.tasks[].owner` — `dev` or `devops`.
**Answer:** `payload.tasks[].taskType` — `feature`, `fix`, `refactor`, `config`, `docs`, or `test`.

### Paragraph 4 — Inputs

**Scope:** The data that the owner needs.

**Questions:**
1. Which form does the owner read?
2. Which `objectId` does the owner change?

**Answer:** `payload.tasks[].inputs[]` — `Ref[]`.
**Answer:** `payload.tasks[].contractRefs[]` — contract IDs from the `LLD`.

### Paragraph 5 — Outputs

**Scope:** The target-system objects that the task creates or changes.

**Questions:**
1. Which existing `objectId` does the task change?
2. Which new object does the task create?
3. Which object does the task delete?

**Answer:** `payload.tasks[].outputs[]` — `TargetRef[]`.

### Paragraph 6 — Acceptance

**Scope:** The condition that proves the task is complete.

**Questions:**
1. Which measurable condition proves completion?
2. Which `fr.id` or `path.id` does the condition follow from?
3. Which test proves the condition?

**Answer:** `payload.tasks[].acceptance[]` — `Statement[]`.
**Answer:** `payload.tasks[].acceptanceRefs[]` — `fr.id` or `path.id` values.

### Paragraph 7 — Dependencies

**Scope:** The tasks that must finish first.

**Questions:**
1. Which task must finish before this task starts?
2. Which task is on the critical path?

**Answer:** `payload.tasks[].dependsOn[]` — `task.id` values.
**Answer:** `payload.criticalPath[]` — ordered `task.id` values.

## Gates

1. Each task has exactly one `owner`.
2. The tree has no cycle.
3. Each `parentId` exists in `tasks`.
4. Each task references at least one contract or one ADR.
5. Each task has at least one acceptance statement.
6. Each acceptance statement has a measurable condition and a test reference.
7. A leaf task covers one testable unit.
8. `criticalPath` is a subset of `tasks`.
9. A delta does not change the `id` of an existing task.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| none | none | The plan references objects through `outputs`. It does not change them. |

## Specificity note

"Works correctly" is not an acceptance condition. The condition must name a value, a unit, and a test.
