# VERDICT — QA Verdict

| field | value |
| --- | --- |
| formType | `VERDICT` |
| stage | `09` |
| owner | `qa` |
| route | `greenfield`, `change` |
| mode | `full` |
| inputs | `TEST_CONTRACT`, `IMPLEMENTATION_SPEC`, test report |
| next | `REFACTOR_SPEC` on fail, or task completion |

## Template

### Paragraph 1 — Run identity

**Scope:** The test run that produced the result.

**Questions:**
1. Which task does the run check?
2. Which implementation does the run check?
3. Which run ID identifies the report?
4. When did the run start and finish?

**Answer:** `payload.taskRef`, `payload.implementationRef`, `payload.runId`, `payload.startedAt`, `payload.finishedAt`.

### Paragraph 2 — Test result

**Scope:** The result of one test.

**Questions:**
1. Which test ran?
2. Which status did the test get?
3. How long did the test run?
4. Which artifact holds the evidence?

**Answer:** `payload.results[]` — `{ testId, status, durationMs, evidence: Ref }`.

### Paragraph 3 — Violation

**Scope:** One rule that the implementation breaks.

**Questions:**
1. Which test shows the violation?
2. Which aspect does the violation break?
3. Which severity applies?
4. Which artifact holds the proof?

**Answer:** `payload.violations[]` — `{ testId?, aspect, severity, evidence: Ref }`.

### Paragraph 4 — Verdict

**Scope:** The decision for the task.

**Questions:**
1. Did all tests pass?
2. Which reason supports the verdict?

**Answer:** `payload.verdict` — `pass` or `fail`.
**Answer:** `payload.reason` — one `Text` value.

### Paragraph 5 — Evidence set

**Scope:** The full evidence for an auditor.

**Questions:**
1. Which log file proves the run?
2. Which coverage report proves the test scope?

**Answer:** `payload.evidence[]` — `Ref[]`.

## Gates

1. The `results` set equals the `TEST_CONTRACT.tests` set for the task.
2. `verdict: pass` requires `status: pass` for every result.
3. `verdict: pass` is invalid with a `blocker` violation.
4. `status: skipped` counts as `fail` without an operator approval.
5. `violations` is empty for `pass`.
6. Each result references an existing evidence artifact.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `test` | read | Read the test registry. |
| `module` | read | Read the implementation. |
| `config` | read | Read the applied configuration. |
| `env` | read | Read the runtime environment. |

## Specificity note

A verdict without a run ID and evidence is a defect.
