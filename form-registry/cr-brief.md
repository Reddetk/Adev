# CR_BRIEF — Change Request Brief

| field | value |
| --- | --- |
| formType | `CR_BRIEF` |
| stage | `00` |
| owner | `analyst` |
| route | `change` |
| mode | `full` |
| inputs | `USER_INPUT`, current artifacts |
| next | `IMPACT_ASSESSMENT` |

## Template

### Paragraph 1 — Change class

**Scope:** The type and the cause of the change.

**Questions:**
1. Which change class applies?
2. Which event caused the change?
3. Which person or system requested the change?

**Answer:** `payload.changeType` — `business`, `devsecops`, `techdebt`, `audit`.
**Answer:** `payload.driver` — one `Text` value.

### Paragraph 2 — Current behavior

**Scope:** The behavior that exists now. Use registry IDs.

**Questions:**
1. Which `objectId` shows the current behavior?
2. Which input starts the current behavior?
3. Which output shows the current behavior?
4. Which error appears now?

**Answer:** `payload.currentBehavior` — `{ objectId, input, output, error? }`.

### Paragraph 3 — Requested behavior

**Scope:** The behavior that must exist after the change.

**Questions:**
1. Which input must start the new behavior?
2. Which output must the new behavior give?
3. Which error must disappear?
4. Which new error can appear?

**Answer:** `payload.requestedBehavior` — `{ input, output, removedError?, newError? }`.

### Paragraph 4 — Affected domains

**Scope:** The bounded contexts that the change touches.

**Questions:**
1. Which bounded context contains the affected object?
2. Which second bounded context consumes the result?

**Answer:** `payload.domains[]` — `boundedContext.id` values from the registry.

### Paragraph 5 — Evidence

**Scope:** Records that prove the need for the change.

**Questions:**
1. Which incident, audit finding, or ticket supports the change?
2. Which file or record holds the evidence?

**Answer:** `payload.evidence[]` — `Ref[]`.

### Paragraph 6 — Urgency

**Scope:** The required start time.

**Questions:**
1. Which urgency level applies?
2. Which date is the latest acceptable start?

**Answer:** `payload.urgency` — `low`, `normal`, `high`, `critical`.
**Answer:** `payload.latestStart` — RFC3339 date.

## Gates

1. `requestedBehavior.output` differs from `currentBehavior.output`.
2. `evidence` is not empty for `changeType: audit` and `devsecops`.
3. Each `domains[]` value exists in the object registry.
4. `urgency: critical` requires an operator note in `driver`.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `container` | read | Read the object registry. |
| `module` | read | Read the current implementation. |
| `config` | read | Read the current configuration. |
| `pipeline` | read | Read the current pipeline. |
| `env` | read | Read the current environment. |
