# IMPLEMENTATION_SPEC — Implementation Spec Form

| field | value |
| --- | --- |
| formType | `IMPLEMENTATION_SPEC` |
| stage | `08` |
| owner | `dev` |
| route | `greenfield`, `change` |
| mode | `full` |
| inputs | `TEST_CONTRACT`, `LLD`, `ADR`, object registry |
| next | `VERDICT` |

## Template

### Paragraph 1 — Task binding

**Scope:** The task and the test contract that the implementation serves.

**Questions:**
1. Which task does the implementation serve?
2. Which test contract defines the expected behavior?

**Answer:** `payload.taskRef` — `task.id`.
**Answer:** `payload.testContractRef` — `Ref`.

### Paragraph 2 — Target list

**Scope:** Each target-system object that the implementation touches.

**Questions:**
1. Which `objectId` does the implementation change?
2. Which action applies to the object?
3. Which object does the implementation create?
4. Which object does the implementation delete?

**Answer:** `payload.targets[]` — `{ target: TargetRef, action: "create" | "change" | "delete", generator }`.

### Paragraph 3 — Generation method

**Scope:** The tool that produces each target.

**Questions:**
1. Which generator produces the target?
2. If no generator can produce the target, which reason applies?
3. Which operator approved the exception?

**Answer:** `payload.targets[].generator` — `codegen`, `configgen`, or `manualException`.
**Answer:** `payload.exception` — `{ reason: Text, approvalRef }` for `manualException`.

### Paragraph 4 — Steps

**Scope:** The ordered actions for one target at a time.

**Questions:**
1. Which step number applies?
2. Which target does the step change?
3. Which action does the step do?

**Answer:** `payload.steps[]` — `{ order, targetObjectId, action, text: Text }`.

### Paragraph 5 — Constraints

**Scope:** The rules that the implementation must obey.

**Questions:**
1. Which ADR decision limits the implementation?
2. Which invariant must stay true?
3. Which dependency is forbidden?

**Answer:** `payload.constraints[]` — `Statement[]`.

### Paragraph 6 — Test mapping

**Scope:** The link from a test to the implemented target.

**Questions:**
1. Which `test.id` proves the target?
2. Which step implements the target?

**Answer:** `payload.testMap[]` — `{ testId, targetObjectId, stepOrder }`.

## Gates

1. Each target has an `objectId` from the registry or `new: true` with `kind` and `path`.
2. `generator: manualException` requires a non-empty `exception.approvalRef` with `status: approved`.
3. `manualException` cannot target `kind: test`.
4. Each `TEST_CONTRACT.tests[]` entry has a `testMap[]` entry.
5. The form requires at least one step. Each step has an order value.
6. A step names one `targetObjectId` and one action.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `module` | create / change / delete | Register `status: planned`. |
| `test` | create / change | Generated from `TEST_CONTRACT`. |
| `config` | create / change | Register `status: planned`. |
| `pipeline` | create / change | Register `status: planned`. |
| `env` | create / change | Register `status: planned`. |
| `table` | create / change | Register `status: planned`. |
| `doc` | create / change | Register `status: planned`. |

## Specificity note

A free-text file path outside `TargetRef` is a defect.
`manualException` is the only path for model-written code. It needs an operator approval.
