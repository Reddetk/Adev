# HLD — High-Level Design (to-be)

| field | value |
| --- | --- |
| formType | `HLD` |
| stage | `03` |
| owner | `architect` |
| route | `greenfield`, `change` |
| mode | `full`, `delta` |
| inputs | `SRS`, `FEASIBILITY`, as-is `HLD`, object registry |
| next | `LLD`, `ADR` (change route) |

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
5. Which PACELC class does this context promise: PA/EL, PC/EC, PA/EC or PC/EL?
6. Which containers implement the context?

**Answer:** `payload.boundedContexts[]` — `{ id, name, purpose: Text, pacelc, containerIds, owns, reason }`.
**Answer:** `pacelc` is `PA/EL`, `PC/EC`, `PA/EC` or `PC/EL`. A context that promises consistency (`PC/...`) needs a consistent store and at least one synchronous edge.

### Paragraph 3 — Container

**Scope:** One deployable unit. The user-facing services, the external systems and
the infrastructure are all containers: a gateway, an identity provider and a message
bus deploy and fail on their own, so they are not a box inside an application.

**Questions:**
1. Which `objectId` identifies the container?
2. Which bounded context does the container implement?
3. Which runtime technology does the container use, from the registry list?
4. Which NFR drives the technology choice?
5. Which container accepts the request at the edge: a reverse proxy or an API gateway?
6. Which container proves who the user is, when the BRD has a sign-up or a sign-in?
7. Which container carries the events, when an operation runs asynchronously?
8. Which container balances the traffic, when more than one container serves the same client?

**Answer:** `payload.containers[]` — `{ objectId, purpose: Text, boundedContextId, tech, nfrRefs }`.
**Answer:** `tech` is a closed list: `go`, `java`, `node`, `python`, `react`, `angular`, `vue`, `postgresql`, `mysql`, `mongodb`, `cassandra`, `redis`, `kafka`, `rabbitmq`, `nats`, `nginx`, `envoy`, `kong`, `keycloak`, `elasticsearch`, `minio`, `kubernetes`, `external`. A product outside the list has no container.
**Answer:** an infrastructure container has its own `boundedContextId`, its own `tech` and its own degradation entry.

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
**Answer:** `sync` is a call that waits for the answer; `async` is an event the bus carries after the caller returns.

### Paragraph 6 — Semantic analysis

**Scope:** Record of the semantic analysis that led to context boundaries.

**Questions:**
1. Which term was analyzed?
2. In which contexts does it appear with different meanings?
3. What are the differences (attributes/lifecycle/behavior)?
4. Why does this require different aggregates/contexts?
5. Which bounded contexts result from this split?

**Answer:** `payload.semanticAnalysis[]` — `{ term, contexts: { boundedContextId, meaning, attributes, lifecycle }[], rationale, result }`.

### Paragraph 7 — Decision

**Scope:** One fork the design settles: which product for the storage, the bus, the
edge, the identity; which consistency; which failure handling. A greenfield system has
no decision history, so its choices live in the design they shape.

**Questions:**
1. Which option does the design take?
2. Which options does it reject, and which SRS requirement, NFR or constraint makes them wrong?
3. Which containers does the decision fix?
4. Which quality does the decision improve, and which does it cost?
5. Which area of the design does the decision belong to: architecture, communication, storage, scaling, consistency, reliability, stack, ops, process or metrics?
6. For a consistency decision: which bounded context does it settle?
7. For an architecture decision: which style, from monolith, modular_monolith, hexagonal, microservices, event_driven, cell_based or serverless?
8. Which distributed-data premise does the design hold: an async write (outbox), an at-least-once delivery (DLQ), an async step through a bus (backpressure), or a chain of relations across services (saga)? Name the mechanism of that branch in `pattern`.
9. Where do the containers run, from multiple_services_per_host, service_per_host, service_per_vm, service_per_container, serverless or service_deployment_platform?

**Answer:** `payload.decisions[]` — `{ id, area, title: Text, decision: Text, context: Text?, alternatives: [{option, rejectedBecause}], consequences: Text[], objectRefs: string[], techs: string[], contextRef?, style?, pattern?, deployment?, checklist? }`.
**Answer:** `objectRefs` names containers of this design. `techs` names the products the decision chooses, and each product must be the `tech` of a container of this design.
**Answer:** one consistency decision per bounded context (`contextRef`), one storage decision per design that has a store container (`ownership`: `database_per_service`, `shared_database` or `read_model`), and exactly one architecture decision (`style`).
**Answer:** `pattern` is one of the D7 mechanisms: `transactional_outbox`, `transaction_log_tailing`, `polling_publisher`, `dlq`, `parking_lot`, `bounded_queue_block`, `drop_new`, `degrade`, `scale_out`, `saga_orchestration`, `saga_choreography`, `two_phase_commit`. A pattern is present only when its premise is present.
**Answer:** exactly one `ops` decision names `deployment`; the system writes its `checklist` from the pattern.

### Paragraph 8 — Degradation strategy (v2)

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
10. A change to `boundedContexts` or `aggregates` requires an `ADR` on the change route.
11. Each decision has at least one rejected alternative, and every `objectRefs` value is a container of this design.
12. Every `decisions[].techs` value is the `tech` of a container of this design: a database, a cache or a bus the system takes is a container, not a name in a sentence.
13. Every `boundedContexts[].pacelc` is `PA/EL`, `PC/EC`, `PA/EC` or `PC/EL`.
14. Every `containers[].tech` is one of the technologies the registry lists.
15. A consistent context (`PC/EC`, `PC/EL`) does not rest only on available-only stores (`cassandra`, `redis`).
16. A consistent context that takes part in relations has at least one synchronous edge.
16a. A storage decision names `ownership`; a style that deploys apart cannot share one database.
16b. An async relation names its `guarantee`, and an at-least-once edge into a consistent context names its `idempotencyKey`.
17. Every `decisions[].area` is one of architecture, communication, storage, scaling, consistency, reliability, stack, ops, process, metrics.
18. Every bounded context has a consistency decision whose `contextRef` names it.
19. A design with a store container has a storage decision whose `techs` names that store.
20. The design has exactly one architecture decision, `style` is one of the seven, and a monolith-like style carries at most one code container while a distributed style carries at least two.
21. An async relation that leaves a container with an aggregate has a decision whose `pattern` is `transactional_outbox`, `transaction_log_tailing` or `polling_publisher`.
22. An async relation with `at_least_once` or `exactly_once` has a decision whose `pattern` is `dlq` or `parking_lot`.
23. An async relation that enters a message bus (`kafka`, `rabbitmq`, `nats`) has a decision whose `pattern` is `bounded_queue_block`, `drop_new`, `degrade` or `scale_out`.
24. A chain of two relations across bounded contexts has a decision whose `pattern` is `saga_orchestration`, `saga_choreography` or `two_phase_commit`.
25. A decision names a `pattern` only when its premise holds; exactly one `ops` decision names `deployment` from the closed list, the serverless style takes the serverless deployment, and its `checklist` is written by the system from the pattern.

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
