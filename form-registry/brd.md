# BRD — Business Requirements Document

| field | value |
| --- | --- |
| formType | `BRD` |
| stage | `00` |
| owner | `analyst` |
| route | `greenfield` |
| mode | `full` |
| inputs | `USER_INPUT` |
| next | `FEASIBILITY` |

## Template

### Paragraph 1 — Business outcome

**Scope:** The business result and the metric that proves it. Do not describe architecture.

**Questions:**
1. Which business result must the target system cause?
2. Which metric measures this result?
3. Which unit does the metric use?
4. What is the baseline value of the metric?
5. What is the target value of the metric?

**Answer:** `payload.goals[]` — `{ id, outcome: Text, metricId }`.
**Answer:** `payload.metrics[]` — `{ id, name, unit, baseline, target }`.

### Paragraph 2 — Project boundary

**Scope:** The capabilities inside and outside the project.

**Questions:**
1. Which capability is inside the project boundary?
2. Which capability is outside the project boundary?
3. Which capability does the project defer to a later release?

**Answer:** `payload.scope` — `{ in: Text[], out: Text[], deferred: Text[] }`.

### Paragraph 3 — Actors and tasks

**Scope:** The people or systems that use the target system.

**Questions:**
1. Which actor starts the use case?
2. Which task does the actor do?
3. How often does the actor do the task?
4. Which business object does the actor change?

**Answer:** `payload.actors[]` — `{ id, name, tasks[], frequency, businessObject }`.

### Paragraph 4 — Constraints and assumptions

**Scope:** Limits that the solution must obey.

**Questions:**
1. Which law or regulation limits the solution?
2. Which budget limit applies?
3. Which deadline applies?
4. Which assumption must be true for the project to succeed?

**Answer:** `payload.constraints[]` — `Statement[]`.
**Answer:** `payload.assumptions[]` — `[Text]`.

### Paragraph 5 — Success measurement

**Scope:** The event and the time window that prove success.

**Questions:**
1. Which metric proves the goal?
2. Which value marks success?
3. Which measurement window applies?

**Answer:** `payload.successMetrics[]` — `{ metricId, threshold, window }`.

### Paragraph 6 — Missing data

**Scope:** Data that the operator must supply.

**Questions:**
1. Which field is absent?
2. Which question must the operator answer?

**Answer:** `payload.openGaps[]` — `Gap[]`.

## Gates

1. `goals` contains at least one entry.
2. Each `goals[].metricId` exists in `metrics`.
3. Each `metrics[]` entry has `unit`, `baseline`, and `target`.
4. No field describes a container, module, or code.
5. The analyst closes all `blocking` gaps before `status: approved`.
6. `BRD-clean` is a new `formVersion` after the analyst removes rejected scope.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| none | none | The BRD describes business data only. |

## Specificity note

`frequency` is an enum: `day`, `week`, `month`, `quarter`, `event`.
`businessObject` is a business noun, not a target-system `objectId`.
