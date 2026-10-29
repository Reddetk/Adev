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
4. In which bounded context does this term have this meaning?

**Answer:** `payload.ubiquitousLanguage[]` — `{ term, definition: Text, sourceRef, boundedContextId? }`.

### Paragraph 2 — Bounded context

**Scope:** One boundary of meaning found by semantic analysis.

**Questions:**
1. Which business capability does the context own?
2. Which lifecycle or actor owns this context?
3. Which terms change meaning across this boundary?
4. Why can these terms not share the same model?
5. Which containers implement the context?

**Answer:** `payload.boundedContexts[]` — `{ id, name, purpose: Text, containerIds, owns, reason }`.

### Paragraph 3 — Container

**Scope:** One deployable unit.

**Questions:**
1. Which `objectId` identifies the container?
2. Which bounded context does the container implement?
3. Which runtime technology does the container use?
4. Which NFR drives the technology choice?

**Answer:** `payload.containers[]` — `{ objectId, purpose: Text, boundedContextId, tech, nfrRefs }`.

### Paragraph 4 — Aggregate

**Scope:** One consistency boundary inside a container found by invariant analysis.

**Questions:**
1. Which `objectId` is the aggregate root?
2. Which container and bounded context owns the aggregate?
3. Which entities belong to the aggregate?
4. Which value objects belong to the aggregate?
5. Which business invariant must this aggregate protect?
6. Which operation would break that invariant?
7. Which event does the aggregate publish?
8. Which module implements the root entity?

**Answer:** `payload.aggregates[]` — `{ id, containerId, boundedContextId, root, entityRefs, valueObjectRefs, invariants: Statement[], moduleRef, eventMaps }`.

### Paragraph 5 — Context relation

**Scope:** One dependency between two contexts.

**Questions:**
1. Which context is the upstream side?
2. Which context is the downstream side?
3. Which contract or event carries the data?
4. Is the call synchronous or asynchronous?
5. What is the reason for this relationship (published language / conformist / ACL)?

**Answer:** `payload.contextRelations[]` — `{ from, to, type: "sync" | "async", contractId?, eventRef?, relationshipType, reason }`.

### Paragraph 6 — Semantic analysis

**Scope:** Record of the semantic analysis that led to context boundaries.

**Questions:**
1. Which term was analyzed?
2. In which contexts does it appear with different meanings?
3. What are the differences (attributes/lifecycle/behavior)?
4. Why does this require different aggregates/contexts?
5. Which bounded contexts result from this split?

**Answer:** `payload.semanticAnalysis[]` — `{ term, contexts: { boundedContextId, meaning, attributes, lifecycle }[], rationale, result }`.

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
3. Each `aggregate.boundedContextId` exists in `boundedContexts`.
4. Each `aggregate.root` has a `moduleRef` in the object registry or a new `TargetRef`.
5. Each `contextRelations[].contractId` or `contextRelations[].eventRef` exists.
6. Each aggregate has at least one invariant that protects its consistency boundary.
7. Each container with an external dependency has a degradation strategy.
8. An aggregate without a root is invalid.
9. Terms that appear in multiple contexts are recorded in `ubiquitousLanguage` with their context-specific meanings.
10. A change to `boundedContexts` or `aggregates` requires an `ADR`.

## Delta

`base` is the approved `HLD`. The `ops` patch changes `/payload/boundedContexts`, `/payload/aggregates`, `/payload/contextRelations`, and `/payload/semanticAnalysis`.
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
Use semantic analysis to find homonyms. Each aggregate protects exactly the invariants it needs; do not create god objects. Use ubiquitous language per context.
