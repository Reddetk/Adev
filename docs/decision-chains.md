# Decision chains: from an event storming of adev to a classifier-driven design

This document answers three questions in one pass: what adev does today, in which
order its design decisions must be made, and which of them a classifier should
take.

Two sources define the palettes, and they do not overlap:

- **[microservices.io/patterns](https://microservices.io/patterns/index.html)** gives
  the **named mechanisms and their guarantees**: Database per Service, Saga,
  Transactional outbox, Idempotent Consumer, Circuit Breaker, API Gateway, Access
  Token. A decision names one of these, not a mood.
- **[system-design.space](https://system-design.space/)** gives the **theory behind
  the fork**: PACELC classes, the database selection framework, backpressure,
  Kafka delivery semantics, Kubernetes operations, consistency models under test.

The rule of the document is **code-first**. Every event names the function that
emits it, every premise names the field that stores it, every decision names the
gate that checks it. A diagram that no struct can carry is not a design.

The failure this work removes is visible in the last live run (pipeline 2, parcel
delivery): the ADR chose a modular monolith over nine deployed services, named a
cache and a search engine that no container had, and cited BRD constraints the
stage never reads. None of that was a model quality problem. It was a missing
premise and a missing palette.

## 0. The catalogues as palettes

| microservices.io category | palette (exact pattern names) | adev node |
|---|---|---|
| Architectural style | monolithic, microservice | D6 |
| Service boundaries | decompose by business capability, by subdomain, self-contained service, service per team | D6 |
| Service collaboration | Database per Service, Shared database, Saga, API Composition, CQRS, Domain event, Event sourcing | D4, D7 |
| Transactional messaging | Transactional outbox, Transaction log tailing, Polling publisher | D7 |
| Communication styles | Remote Procedure Invocation, Messaging, domain-specific protocol, Idempotent Consumer | D5 |
| External API | API gateway, Backend for front-end | D5 |
| Service discovery | client-side, server-side, service registry, self/3rd-party registration | D8 |
| Reliability | Circuit Breaker | D7 |
| Security | Access Token | D8 |
| Observability | log aggregation, application metrics, audit logging, distributed tracing, exception tracking, health check API | D9 |
| Deployment | multiple services per host, service per host, per VM, per container, serverless, service deployment platform | D8 |
| Cross-cutting | microservice chassis, externalized configuration, service template | D3, D8 |
| Testing | consumer-driven contract test, consumer-side contract test, service component test | stage 07 |
| Refactoring to services | Strangler Application, anti-corruption layer | change route |

The theory catalogue decides **which** option is right; the pattern catalogue
decides **what the option is called** and what it guarantees.

## 1. Event storming of adev as it runs today

### 1.1 The pipeline

| event | emitted in | stored in |
|---|---|---|
| `RequestSubmitted` | `App.tsx submitInput` → `POST /forms` | `forms` row `USER_INPUT`, `status: draft` |
| `PipelineStarted` | `App.tsx createPipeline` → `pipelineService.CreatePipeline` | `pipelines`, `status: ready`, `current_stage: 00` |
| `StageStarted` | `StageRunService.Start` | `runs`, `status: running` |
| `StageChained` | `Orchestrator.Run` → `domain.NextStage` | `pipelines.current_stage` moves |
| `PipelineCompleted` | `Orchestrator.Run`, last stage of the route | `pipelines.status: completed` |

### 1.2 One stage run

| event | emitted in | effect |
|---|---|---|
| `ContextBuilt` | `StageService.buildContext` | inputs projected by role, refs collected |
| `ModelCalled` | `startModelCall` / `finishModelCall` | `model_calls` row with prompt and answer |
| `SkeletonStored` | `detailBatches`, first checkpoint | draft with the outline only |
| `EntryDetailed` | `detailLevel` merge + `progressSaver` | draft rewritten with one more entry |
| `GatesPassed` | `FormService.Submit` | `forms.status: valid` |
| `GatesFailed` | `SubmitDraft` + `record("blocked")` | draft kept, `runs.last_error` names the failure |
| `StageStopped` | `runForm`, `ctx.Err()` branch | draft saved on a non-cancelled context, `record("stopped")` |

### 1.3 The question loop

| event | emitted in | stored in |
|---|---|---|
| `QuestionPublished` | `PublishGaps` / `PublishGate` | `operator_requests`, at most 5 per run |
| `QuestionAnswered` | `QuestionService.Answer` | draft payload patched in place |
| `QuestionClosedStale` | `CloseStale` after a valid version | stale requests closed instead of blocking |

### 1.4 Policies of the current code

| when | then | code |
|---|---|---|
| gates failed on HLD or LLD, attempt < 3 | retry the whole document with the failures | `structuralRetry` |
| a gap names one answerable value | publish it, hold the stage | `askableGap`, `questionsPerRun` |
| the operator stopped the agent | save the draft, report `stopped` | `progressSaver`, stop branch of `runForm` |
| a decision names a container or a product | require that container and its `tech` to exist | `ValidateHLD` decisions gate |
| the form is valid | the next stage starts by itself | `Orchestrator.Run` |

### 1.5 Hot spots: where the design leaks today

| id | leak | where |
|---|---|---|
| H1 | `tech` is free text, so the LLD path root is an interpretation of a sentence | `Container.Tech`, `detailInstruction "packages"` |
| H2 | a decision proves a container exists, not that the mechanism fits the workload | `ValidateHLD` |
| H3 | no premises: no QPS, storage, read/write ratio, team, stack or SLO | `Feasibility`, `FunctionalRequirement` |
| H4 | module `deps` are never checked against HLD relations | `ValidateLLD` |
| H5 | LLD contract ids are not tied to the HLD `contractId` | `Contract`, `ContextRelation` |
| H6 | the platform has no field, so its checklist cannot be conditional | nowhere |
| H7 | every choice is prose: no pattern name, no guarantee, no confidence, no author | `Decision`, `Option` |

## 2. The decision chain

Each node reads only the premises its parents produced. A node whose premise is
missing does not run: it becomes a question, never a default.

```
D0 premises ──► D1 PACELC per context ──► D2 failure profile ──► D5 communication
   │                  │                          │                    │
   │                  └────────► D6 architecture ◄┴──────────► D7 distributed data
   │                                  │                             │
   └──► D3 language ──► D4 storage + data ownership ────────────────┘
                                            │
                                            ▼
                                     D8 platform ──► D9 observability
```

### D0. Premises

Nothing here is a choice. A premise is a fact the operator or the code already
knows.

| premise | form | field today | field to add |
|---|---|---|---|
| peak and average QPS, read/write ratio | TEJ | none | `premises.traffic.peakQps`, `readWriteRatio` |
| payload size, bandwidth | TEJ | none | `premises.traffic.payloadBytes` |
| data volume per year, growth | TEJ | none | `premises.storage.gbPerYear`, `growth` |
| team size, known languages, ops maturity | TEJ | none | `premises.team` |
| monthly budget, deadline | TEJ | `options[].cost`, `scheduleDays` | `premises.budgetMonthly`, `deadlineDays` |
| availability target, RPO, RTO | TEJ | none | `premises.availability`, `rpoMinutes`, `rtoMinutes` |
| laws, data classes | BRD | `constraints`, `glossary` | keep |
| concurrency per operation: two callers, one key | SRS | none | `fr.concurrency`, `fr.idempotency` |
| delivery guarantee of an async step | SRS | none | `fr.delivery`, `fr.failureMode` |
| who calls whom, with what guarantee | HLD | `contextRelations` | `guarantee`, `idempotencyKey`, `retry`, `dlq` |

Without D0 a classifier sees "the system processes parcels" and picks PostgreSQL
because the sentence is about records.

### D1. PACELC per bounded context, then the system

Decided first, because it constrains every later node. The theory gives four
classes, and that closed set is the whole palette:

| class | inside a partition | in the normal case | fits |
|---|---|---|---|
| `PA/EL` | availability over consistency | latency over consistency | feeds, catalogs |
| `PC/EC` | consistency over availability | consistency over latency | money, inventory, booking |
| `PA/EC` | availability over consistency | consistency over latency | reads over a consistent store |
| `PC/EL` | consistency over availability | latency over consistency | latency-critical writes with read repair |

- decider: classifier. Four options, numeric premises.
- per context first, then a system summary: the system class summarizes the
  context classes, it does not vote. A parcel system can be `PC/EC` for payment
  and `PA/EL` for tracking.
- gate: every bounded context has a class, the class is one of the four, and a
  `PC/EC` context cannot get an AP-only store or an async-only relation.
- lands in: `HLD.boundedContexts[].pacelc` and a `decisions[]` entry with
  `techs: []` and `objectRefs` naming the context's containers.
- evidence: an NFR row (consistency or availability metric) or a BRD constraint.

### D2. Failure profile

Derived by code from D1 and the NFRs: `rpoMinutes`, `rtoMinutes`, retry budget and
the consistency model the SRS promises (linearizable, serializable,
read-your-writes, eventual). This profile feeds D5 and D7 and is stored once.

### D3. Language and runtime

Tool first, product second:

1. `premises.team.languages` comes from the operator;
2. the system takes that language as a rule; the palette holds one option, and the
   decision records the premise as its evidence;
3. only an explicit operator request to learn a language opens a second option, and
   then it is a TEJ cost, not a style preference;
4. the runtime scaffolding is the **microservice chassis** and the **service
   template** of the pattern catalogue: one answer for logging, tracing, config
   and health, or the platform pays for it per service.

The LLD path root follows a table in code (`go → internal/<layer>/<module>`,
`java → src/main/java/<module>/<layer>`, `ts → src/<layer>/<module>`), so H1
disappears.

### D4. Storage and data ownership

The database selection framework separates the workload from the engine
(OLTP/OLAP, read/write ratio, consistency, replication, sharding, TCO). The chain:

1. **mechanism**: transactions and joins, access by key, wide rows with
   aggregation, graph closure, object bytes, full text, time series;
2. **class**: relational, key-value/in-memory, document, wide-column, search,
   time-series, graph, vector, object;
3. **product**, filtered by budget, team stack and licence;
4. **ownership**: Database per Service, Shared database, or a read model
   (CQRS) — a decision of its own, because it changes every later relation.

Only step 3 is a classifier. The product becomes a container with its `tech`,
which the existing gate already requires. Replication and sharding follow the
premises: leader-based, leaderless with `R + W > N`, or none; consistent hashing
when a hot key is plausible.

### D5. Communication

Per relation, from D1 and D2. The pattern catalogue gives the style and the theory
gives the guarantee:

| style | pattern | guarantee | cost |
|---|---|---|---|
| sync | Remote Procedure Invocation over REST | request/response, simple contracts | chatty, latency per hop |
| sync | RPI over gRPC | typed contract, streaming | schema tooling |
| sync | RPI over GraphQL | client-shaped reads | query complexity limits |
| async | Messaging: queue | at-least-once, leveling | consumer idempotency |
| async | Messaging: pub/sub topic | fan-out, at-least-once | ordering per key |
| async | Messaging: event stream | replay, ordered per partition | operational weight |
| any | Idempotent Consumer | duplicate delivery is safe | a key and a store |
| external | API gateway / Backend for front-end | one entry per client kind | one more hop |

An async relation without an Idempotent Consumer is a design that loses money on
retries; the gate refuses it when the SRS says a retry is possible.

### D6. Architecture style

The last big choice, not the first. Premises: the TEJ verdict and budget, the SRS
NFRs, the bounded contexts from the HLD and D1 to D5.

- palette: monolith, microservices; then the boundary pattern that justifies it —
  decompose by business capability, by subdomain, self-contained service, service
  per team.
- decider: classifier for the style, because the premises are numeric (team size,
  budget, SLO, number of contexts).
- gate: the style must match the container count and the relation types. Nine
  deployed services are not a monolith; one deployable unit is not microservices.
  A microservice style must name its boundary pattern; a monolith must name the
  module boundary instead.
- evidence: `feasibility.recommendedOption`, `feasibility.premises`, `srs.nfr[]`.

### D7. Distributed data and reliability

A tree, not one choice. Each branch is skipped when its premise is absent, and
each answer is a named pattern with its guarantee:

| premise | question | palette (pattern names) | guarantee |
|---|---|---|---|
| two callers reach one key at once | concurrency control | optimistic version, pessimistic lock, single writer | no lost update vs no blocking |
| a write and an event must agree | transactional messaging | Transactional outbox, Transaction log tailing, Polling publisher, none | atomic state change and publish |
| one operation spans three services | consistency across services | Saga (orchestration), Saga (choreography), 2PC | completion without a global lock |
| a query needs data from three services | query shape | API Composition, CQRS read model | cheap reads vs stale reads |
| the audit trail is a requirement | event history | Event sourcing, domain event + log | full history vs mutable state |
| a consumer sees a duplicate | duplicate handling | Idempotent Consumer, dedup table, none | safe retries |
| a call can time out | failure isolation | Circuit Breaker, retry + backoff, bulkhead | no cascade |
| a message fails forever | poison handling | DLQ, parking lot, none | isolation vs silent loss |
| a producer outruns a consumer | flow control | bounded queue + block, drop, degrade, scale | no unbounded memory |
| a lock must survive a pause | safety | lease + fencing token | stale owner cannot write |

Kafka-specific branches: producer acks and `min.insync.replicas` decide whether a
write survives a broker loss; consumer groups decide parallelism; ordering exists
per partition, so the partition key is a design decision; delivery is
at-least-once in practice, and exactly-once is a consumer-side effect bought with
Idempotent Consumer or an idempotent sink.

### D8. Platform and cross-cutting concerns

Deployment patterns from the catalogue: multiple services per host, service per
host, service per VM, service per container, serverless, service deployment
platform. The chosen one defines its checklist:

- **Kubernetes**: one deployment per container, replica floor and ceiling, pod
  disruption budget, readiness and liveness probes backed by a real endpoint,
  resource requests and limits, autoscaling on a real metric, externalized
  configuration, secrets injection, network policy between contexts.
- **VM**: systemd unit, log rotation, disk and backup, rolling restart.
- **Serverless**: cold start budget, concurrency limit, payload size, state
  ownership, idempotency on retries.

Service discovery joins here: client-side or server-side discovery, a service
registry, self or third-party registration. Security joins here as an Access Token
between services, not as a per-service password.

### D9. Observability and success metrics

Closes the loop with the catalogue's observability set: log aggregation,
application metrics, audit logging, distributed tracing, exception tracking,
health check API. Metrics: p99, availability SLO, throughput, error rate, plus the
business metric of the BRD goal.

## 3. What the classifier decides, and what it must not

| node | decider | why |
|---|---|---|
| D1 PACELC class | classifier | four options, numeric premises |
| D4 product inside a class | classifier | closed list, budget filter |
| D6 architecture style | classifier | two styles, numeric premises |
| D7 pattern per branch | classifier | one branch, two or three options |
| D8 deployment pattern | classifier | closed list, operational premises |
| the sentence that justifies a rejection | LLM | must quote the requirement |
| invariants, contracts, module paths | LLM or code | language, not selection |

Mechanics:

1. the palette lives in the registry and in `domain` as a closed list of pattern
   names, so the prompt and the gate read one source;
2. a hard filter in Go removes excluded options (budget, language, licence,
   existing tech) before the model sees them;
3. scoring is `argmax` over what is left, with `{option, confidence}` stored;
4. below a threshold the decision becomes an operator question with the same
   options, and the answer is stored with `source: operator`;
5. every decision records the author (`classifier`, `llm`, `operator`) and the
   confidence.

## 4. Code-first plan

1. **Premises (D0).** `premises` on `Feasibility`, per-FR fields on
   `FunctionalRequirement`, questions in stages 01 and 02, gates. No model needed.
2. **Closed tech list and the path table (H1).**
3. **PACELC per context (D1)**, with the four-class palette and the gate that
   pairs it with the store and the relation.
4. **Data ownership (D4.4) and architecture style (D6).** Two decisions whose
   premise is the TEJ verdict and the NFR rows; the gate checks style against the
   container count.
5. **Communication guarantees (D5) and `deps` ↔ relations (H4).**
6. **Distributed data tree (D7)**: one decision per branch that fired.
7. **Deployment and cross-cutting (D8)**: a deployment pattern, then its checklist.
8. **Classifier adapter** behind the same palette, starting with D1 and D6;
   measure agreement against the decisions already stored.

Stage 07 (test contract) gains the catalogue's testing patterns: consumer-driven
contract test, consumer-side contract test, service component test. The change
route gains the refactoring patterns: Strangler Application, anti-corruption
layer.

## 5. Anti-goals

- A classifier before a palette, a hard filter and a threshold.
- One CAP choice for the whole system; the class is per context.
- Free-text technology anywhere; a product no container carries is a promise.
- A Kubernetes checklist in a VM design, or a saga in a system with one container.
- A decision without an evidence link; "best practice" is not evidence.
- Classifying the reason for a rejection; that is a sentence about a requirement.

## Sources

- [microservices.io/patterns](https://microservices.io/patterns/index.html) —
  the pattern names and their guarantees (styles, boundaries, service
  collaboration, transactional messaging, communication, deployment, reliability,
  security, observability, testing, refactoring).
- [system-design.space](https://system-design.space/) — the theory behind each
  fork: [CAP](/chapter/cap-theorem/) and [PACELC](/chapter/pacelc-theorem/)
  classes, [Jepsen consistency models](/chapter/jepsen-consistency/),
  [database selection framework](/chapter/database-selection-framework/),
  [leaderless replication and `R+W>N`](/chapter/leaderless-replication/),
  [consistent hashing and hot keys](/chapter/consistent-hashing-hot-keys-rebalancing/),
  [saga](/chapter/saga-pattern/), [reliable event delivery](/chapter/reliable-event-delivery/),
  [backpressure](/chapter/backpressure-flow-control/),
  [fencing tokens](/chapter/distributed-locks-leases-fencing/),
  [Kafka guarantees](/chapter/kafka-book/),
  [cloud native and Kubernetes](/theme/cloud-native-containerization/).
