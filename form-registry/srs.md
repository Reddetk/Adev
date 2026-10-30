# SRS — Software Requirements Specification

| field | value |
| --- | --- |
| formType | `SRS` |
| stage | `02` |
| owner | `analyst` |
| route | `greenfield`, `change` |
| mode | `full`, `delta` |
| inputs | `BRD` (clean) and `FEASIBILITY`, or `CR_BRIEF` and `IMPACT_ASSESSMENT` |
| next | `HLD` |

## Template

### Paragraph 1 — Functional requirement

**Scope:** One observable behavior of the target system: one step of the real user flow
of one actor. The sign-up, the sign-in, the cabinet action, the payment and the
notification are steps of a flow, so each one is a requirement of its own.

**Questions:**
1. Which actor starts the behavior?
2. Which input starts the behavior?
3. Which output does the behavior give?
4. Which state changes after the behavior?
5. Which priority does the requirement have?

**Answer:** `payload.fr[]` — `{ id, statement: Statement, actor, trigger, response, stateChange, priority, traceGoal }`.
**Answer:** `fr[].statement` names the actor as the subject: "The client signs in with a phone code".

### Paragraph 2 — Acceptance scenario

**Scope:** One testable path for one functional requirement.

**Questions:**
1. Which `fr.id` does the scenario test?
2. Which state exists before the action?
3. Which action does the actor do?
4. Which result must appear?

**Answer:** `payload.fr[].acceptance[]` — `Scenario[]`.

### Paragraph 3 — Error and boundary behavior

**Scope:** The behavior at an invalid input or a limit.

**Questions:**
1. Which `fr.id` has the error path?
2. Which input is invalid?
3. Which error code must the system return?
4. Which limit value applies?

**Answer:** `payload.fr[].errors[]` — `{ code, condition, limit? }`.

### Paragraph 4 — Non-functional requirement

**Scope:** One measurable quality of the target system.

**Questions:**
1. Which quality does the NFR limit?
2. Which metric measures the quality?
3. Which unit does the metric use?
4. Which threshold must the system meet?
5. Where does the measurement occur?
6. Which time window applies?
7. Which method verifies the value?

**Answer:** `payload.nfr[]` — `{ id, statement: Statement, metric, unit, threshold, measurementPoint, window, verifyMethod }`.

### Paragraph 5 — Out of scope

**Scope:** The behavior that this SRS does not require.

**Questions:**
1. Which behavior is explicitly excluded?
2. Which BRD scope item does the exclusion follow from?

**Answer:** `payload.outOfScope[]` — `{ text: Text, brdScopeRef }`.

### Paragraph 6 — Traceability

**Scope:** The link from each requirement to a goal or a blast-radius item.

**Questions:**
1. Which goal does each FR serve?
2. Which `blastRadius` item does each changed FR serve?

**Answer:** `payload.traceToGoals[]` — goal IDs.
**Answer:** `payload.fr[].traceGoal` — one goal ID.

## Gates

1. At least one of `fr` or `nfr` has an entry.
2. Each `fr` has at least one `acceptance` scenario with non-empty `given`, `when`, and `then`.
3. Each `nfr` has `metric`, `unit`, `threshold`, `measurementPoint`, and `window`.
4. Each `fr.traceGoal` exists in the BRD.
5. Each `Statement.text` passes the ACE check.
6. A delta does not reuse a removed `fr.id`.
7. The `fr` list covers the flow of every actor of the BRD: a sign-in or sign-up, a
   cabinet action, and one requirement for each task the BRD names. A role with no
   requirement is a role the design drops.

## Delta

`mode: "delta"`. `base` is the approved `SRS`. The `ops` patch changes `/payload/fr` and `/payload/nfr` only.
The materialized payload passes the same gates as a full form.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| none | none | The SRS defines behavior, not physical objects. |

## Specificity note

An NFR without `measurementPoint` and `window` is not testable. The form rejects it.
