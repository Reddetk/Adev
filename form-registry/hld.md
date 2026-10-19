# HLD — High-Level Design (to-be)

| field | value |
| --- | --- |
| formType | `HLD` |
| stage | `03` (v1), `04` (v2) |
| owner | `architect` |
| route | `greenfield`, `change` |
| mode | `full`, `delta` |
| inputs | `SRS`, as-is `HLD`, object registry |
| next | `ADR`, `LLD` |

## Template

### Paragraph 1 — Ubiquitous language

**Scope:** The terms that the design and the code use.

**Questions:**
1. Which term has a special meaning in this system?
2. Which definition does the term have?
3. Which SRS requirement defines the term?

**Answer:** `payload.ubiquitousLanguage[]` — `{ term, definition: Text, sourceRef }`.

### Paragraph 2 — Bounded context

**Scope:** One boundary of meaning.

**Questions:**
1. Which business capability does the context own?
2. Which containers implement the context?
3. Which term changes meaning across this boundary?

**Answer:** `payload.boundedContexts[]` — `{ id, name, purpose: Text, containerIds }`.

### Paragraph 3 — Container

**Scope:** One deployable unit.

**Questions:**
1. Which `objectId` identifies the container?
2. Which bounded context does the container implement?
3. Which runtime technology does the container use?
4. Which NFR drives the technology choice?

**Answer:** `payload.containers[]` — `{ objectId, purpose: Text, boundedContextId, tech, nfrRefs }`.

### Paragraph 4 — Aggregate

**Scope:** One consistency boundary inside a container.

**Questions:**
1. Which `objectId` is the aggregate root?
2. Which container owns the aggregate?
3. Which entities belong to the aggregate?
4. Which event does the aggregate publish?
5. Which module implements the root entity?

**Answer:** `payload.aggregates[]` — `{ id, containerId, root, entityRefs, eventMaps, moduleRef }`.

### Paragraph 5 — Context relation

**Scope:** One dependency between two contexts.

**Questions:**
1. Which context is the upstream side?
2. Which context is the downstream side?
3. Which contract carries the data?
4. Is the call synchronous or asynchronous?

**Answer:** `payload.contextRelations[]` — `{ from, to, type: "sync" | "async", contractId? }`.

### Paragraph 6 — Invariant (v2)

**Scope:** One rule that must stay true inside an aggregate.

**Questions:**
1. Which aggregate owns the invariant?
2. Which condition must stay true?
3. Which operation can break the condition?

**Answer:** `payload.aggregates[].invariants[]` — `Statement[]`.

### Paragraph 7 — Degradation strategy (v2)

**Scope:** One failure and the reduced behavior.

**Questions:**
1. Which dependency can fail?
2. Which behavior does the system keep during the failure?
3. Which signal detects the failure?
4. Which timeout applies?

**Answer:** `payload.degradationStrategy[]` — `{ failure, degradation, detection, timeoutMs }`.

## Gates

1. Each `container.boundedContextId` exists in `boundedContexts`.
2. Each `aggregate.containerId` exists in `containers`.
3. Each `aggregate.root` has a `moduleRef` in the object registry or a new `TargetRef`.
4. Each `contextRelations[].contractId` exists in the `LLD` or is new in this form.
5. v2: each aggregate has at least one invariant.
6. v2: each container with an external dependency has a degradation strategy.
7. An aggregate without a root is invalid.
8. A change to `boundedContexts` or `aggregates` requires an `ADR`.

## Delta

`base` is the approved `HLD`. The `ops` patch changes `/payload/boundedContexts`, `/payload/aggregates`, and `/payload/contextRelations`.
A delta cannot remove a container that the current `LLD` uses without an `ADR`.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `container` | create / change | Register each container with `status: planned`. |
| `broker` | read | Read the topic registry for the event map. |
| `doc` | read | Read the as-is HLD. |

`boundedContext` and `aggregate` are fields of a container. They are not separate target objects.

## Specificity note

`tech` must be an ID from the approved technology list. A free-text stack name is a defect.
`timeoutMs` is an integer in milliseconds, not a duration string.
