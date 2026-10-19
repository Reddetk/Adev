# TEST_CONTRACT — Test Contract Form

| field | value |
| --- | --- |
| formType | `TEST_CONTRACT` |
| stage | `07` |
| owner | `qa` |
| route | `greenfield`, `change` |
| mode | `full` |
| inputs | `LLD`, `TASK_PLAN`, object registry |
| next | `IMPLEMENTATION_SPEC` |

## Template

### Paragraph 1 — Test identity

**Scope:** One test for one task and one contract.

**Questions:**
1. Which test ID identifies the test?
2. Which task does the test belong to?
3. Which contract does the test check?

**Answer:** `payload.tests[]` — `{ id, taskRef, contractRef }`.

### Paragraph 2 — Test target

**Scope:** The physical test object.

**Questions:**
1. Which `objectId` identifies the test file?
2. Which test kind applies?
3. Which package holds the test file?

**Answer:** `payload.tests[].targetTest` — `TargetRef`.
**Answer:** `payload.tests[].kind` — `unit`, `contract`, or `integration`.

### Paragraph 3 — Given

**Scope:** The state before the action.

**Questions:**
1. Which object is in the test?
2. Which field value does the object have?
3. Which dependency is a stub or a fake?

**Answer:** `payload.tests[].scenario.given[]` — `[Text]`.

### Paragraph 4 — When

**Scope:** The single action under test.

**Questions:**
1. Which operation does the test call?
2. Which input does the test send?

**Answer:** `payload.tests[].scenario.when[]` — `[Text]`.

### Paragraph 5 — Then

**Scope:** The observable result.

**Questions:**
1. Which output value must appear?
2. Which state must change?
3. Which side effect must not occur?

**Answer:** `payload.tests[].scenario.then[]` — `[Text]`.

### Paragraph 6 — Negative path

**Scope:** The error behavior of the contract.

**Questions:**
1. Which invalid input does the test send?
2. Which error code must the contract return?

**Answer:** `payload.tests[].path` — `happy` or `negative`.
**Answer:** `payload.tests[].expectedErrorCode` — an error code from the contract.

### Paragraph 7 — Coverage

**Scope:** The contracts and paths that the tests cover.

**Questions:**
1. Which contract IDs does the test set cover?
2. Which path IDs does the test set cover?
3. Which contract or path has no test?

**Answer:** `payload.coverage` — `{ contractsCovered[], pathsCovered[], uncovered[] }`.

### Paragraph 8 — Readiness

**Scope:** The condition for the start of implementation.

**Questions:**
1. Are all paths covered?
2. Is the test set ready for implementation?

**Answer:** `payload.ready` — boolean.

## Gates

1. Each `contractRef` exists in the `LLD` and belongs to the task.
2. `coverage.uncovered` is empty if and only if `ready` is `true`.
3. Each task path, happy and negative, has at least one test.
4. Each test has non-empty `given`, `when`, and `then`.
5. A negative test uses an `expectedErrorCode` from its contract.
6. A test does not name an internal function outside `TargetRef`.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `test` | create / change | Register the test file with `status: planned`. |
| `module` | read | Read the contract under test. |

## Specificity note

"Returns an error" is not a test. The test names the error code.
A test without `targetTest.objectId` or `targetTest.new` is a defect.
