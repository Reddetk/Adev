# QUESTION — Operator Question

| field | value |
| --- | --- |
| formType | `QUESTION` |
| stage | `—` (any) |
| owner | any |
| route | `greenfield`, `change` |
| mode | `full` |
| inputs | the form with the missing data |
| next | the interrupted stage |

## Template

### Paragraph 1 — Blocked form

**Scope:** The form and the exact field that lacks data.

**Questions:**
1. Which form has the missing data?
2. Which JSON Pointer identifies the field?
3. Which paragraph of the form does the field belong to?

**Answer:** `payload.formRef` — `Ref`.
**Answer:** `payload.pointer` — JSON Pointer.
**Answer:** `payload.paragraph` — paragraph number.

### Paragraph 2 — Question

**Scope:** One question that the operator can answer.

**Questions:**
1. Which data must the operator supply?
2. Which fact makes the data necessary?
3. Which value does the field accept?

**Answer:** `payload.gaps[].question` — one `Text` value.
**Answer:** `payload.gaps[].fieldType` — enum or atom type.

### Paragraph 3 — Answer options

**Scope:** The allowed answers.

**Questions:**
1. Which option can the operator select?
2. Which option is the default?
3. Which option stops the pipeline?

**Answer:** `payload.gaps[].options[]` — `{ value, label: Text, isDefault }`.

### Paragraph 4 — Blocking state

**Scope:** The effect of the gap on the pipeline.

**Questions:**
1. Does the gap block the next stage?
2. Which stage waits for the answer?

**Answer:** `payload.gaps[].blocking` — boolean.
**Answer:** `payload.waitingStage` — stage ID.

### Paragraph 5 — Wait record

**Scope:** The time and the owner of the wait.

**Questions:**
1. When did the wait start?
2. Which operator must answer?

**Answer:** `payload.waitingSince` — RFC3339.
**Answer:** `payload.assignee` — operator ID.

## Gates

1. `formRef` resolves to an existing form with `status: valid` or `draft`.
2. Each gap has a JSON Pointer to a real field.
3. Each gap has at least one option.
4. `blocking: true` stops all stage transitions for the pipeline.
5. The operator answer creates a new version of the blocked form. It does not edit `QUESTION`.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| none | none | The question reads no target-system object. |

## Specificity note

"Do we need this?" is not a valid question. The question names the field, the fact, and the allowed values.
