# IMPACT_ASSESSMENT — Impact Assessment and Routing

| field | value |
| --- | --- |
| formType | `IMPACT_ASSESSMENT` |
| stage | `01` |
| owner | `analyst` |
| route | `change` |
| mode | `full` |
| inputs | `CR_BRIEF`, current `HLD`, `ADR`, `LLD`, `SRS` |
| next | `SRS` |

## Template

### Paragraph 1 — Blast radius

**Scope:** Each target-system object that the change touches.

**Questions:**
1. Which `objectId` does the change touch?
2. Which action does the change apply to the object?
3. Which depth does the change reach?
4. Which behavior changes at this object?

**Answer:** `payload.blastRadius[]` — `{ objectId, changeKind, depth, reason: Text }`.

### Paragraph 2 — Contract and event impact

**Scope:** The interfaces that break, change, or appear.

**Questions:**
1. Which contract ID changes?
2. Which event name changes?
3. Which consumer stops working after the change?
4. Which compatibility rule applies?

**Answer:** `payload.contractImpact[]` — `{ contractId, eventName?, changeType, consumers, compatibility }`.

### Paragraph 3 — NFR baseline delta

**Scope:** The measured non-functional values before and after the change.

**Questions:**
1. Which NFR ID changes?
2. What is the current value and unit?
3. What is the required value and unit?
4. Which measurement method proves the new value?

**Answer:** `payload.nfrBaselineDelta[]` — `{ nfrId, current, required, unit, verifyMethod }`.

### Paragraph 4 — Routing decision

**Scope:** The route that the change must take.

**Questions:**
1. Which route applies?
2. Which rule ID selects the route?
3. Which fact triggers the rule?

**Answer:** `payload.routing` — `{ route: "A" | "B", ruleId, reason: Text }`.

### Paragraph 5 — Risk

**Scope:** The risks that the change creates.

**Questions:**
1. Which event can cause a failure after the change?
2. What is the likelihood and the impact of the event?
3. Which action reduces the risk?

**Answer:** `payload.risks[]` — `Risk[]`.

## Gates

1. Each `blastRadius[].objectId` exists in the object registry.
2. `route A` is invalid when `depth` is `container`, `context`, or `env`.
3. `route B` is mandatory when a bounded context, an aggregate, or a security contract changes.
4. `routing.ruleId` exists in the routing rulebook.
5. Each `contractImpact[].contractId` exists in the current `LLD`.
6. `nfrBaselineDelta` has `unit` for each entry.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `container` | read | Read the registry and the as-is HLD. |
| `layer` | read | Read the current LLD. |
| `module` | read | Read the current LLD and contracts. |
| `test` | read | Read the current test registry. |
| `config` | read | Read the current configuration. |
| `pipeline` | read | Read the current pipeline. |
| `env` | read | Read the current environment. |
| `broker` | read | Read the current event map. |
| `table` | read | Read the current schema. |
| `doc` | read | Read the ADR and the HLD. |

The form reads the target system. It does not write to it.

## Specificity note

A blast-radius entry without `objectId` is a defect. "The whole system" is not a valid answer.
