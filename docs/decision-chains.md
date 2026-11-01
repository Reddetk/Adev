# Decision chains: from an event storming of adev to a classifier-driven design

This document answers three questions in one pass: what adev does today, in which
order its design decisions must be made, and which of them a classifier should
take. The palettes are not invented here: they follow the taxonomy of
[system-design.space](https://system-design.space/) so that a decision names a
pattern, a mechanism and the guarantee it buys, not a mood.

The rule of the document is **code-first**. Every event names the function that
emits it, every premise names the field that stores it, every decision names the
gate that checks it. A diagram that no struct can carry is not a design.

The failure this work removes is visible in the last live run (pipeline 2, parcel
delivery): the ADR chose a modular monolith over nine deployed services, named a
cache and a search engine that no container had, and cited BRD constraints the
stage never reads. None of that was a model quality problem. It was a missing
premise and a missing palette.

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
| `ModelCalled` | `startModelCall` / `finishModelCall` | `model_calls` row with the full prompt and answer |
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
| H2 | a decision proves a container exists, not that the product fits the workload | `ValidateHLD` |
| H3 | no premises: no QPS, storage, read/write ratio, team, stack or SLO | `Feasibility`, `FunctionalRequirement` |
| H4 | module `deps` are never checked against HLD relations | `ValidateLLD` |
| H5 | LLD contract ids are not tied to the HLD `contractId` | `Contract`, `ContextRelation` |
| H6 | the platform has no field, so the k8s checklist cannot be conditional | nowhere |
| H7 | every choice is prose: no option set, no guarantee, no confidence, no author | `Decision`, `Option` |

## 2. The decision chain

Each node reads only the premises its parents produced. A node whose premise is
missing does not run: it becomes a question, never a default. The order follows
the site: consistency classes first, then storage and communication, then the
style, then the patterns that only make sense inside that style.

```
D0 premises ──► D1 PACELC per context ──► D2 failure profile ──► D5 communication
   │                  │                          │                    │
   │                  └────────► D6 architecture ◄┴──────────► D7 reliability patterns
   │                                  │                             │
   └──► D3 language ──► D4 storage class ──► product ───────────────┘
                                            │
                                            ▼
                                     D8 platform ──► D9 observability
```

### D0. Premises

Nothing here is a choice. A premise is a fact the operator or the code already
knows. The vocabulary is the site's "design process: requirements and estimate"
([]{#cite-1}):

| premise | form | field today | field to add |
|---|---|---|---|
| peak and average QPS, read/write ratio | TEJ | none | `premises.traffic.peakQps`, `readWriteRatio` |
| payload size and bandwidth | TEJ | none | `premises.traffic.payloadBytes` |
| data volume per year, growth | TEJ | none | `premises.storage.gbPerYear`, `growth` |
| team size, known languages, ops maturity | TEJ | none | `premises.team` |
| monthly budget, deadline | TEJ | `options[].cost`, `scheduleDays` | `premises.budgetMonthly`, `deadlineDays` |
| availability target, RPO, RTO | TEJ | none | `premises.availability`, `rpoMinutes`, `rtoMinutes` |
| laws and data classes | BRD | `constraints`, `glossary` | keep |
| concurrency per operation: two callers, one key | SRS | none | `fr.concurrency`, `fr.idempotency` |
| delivery guarantee for an async step | SRS | none | `fr.delivery`, `fr.failureMode` |
| who calls whom, and with what guarantee | HLD | `contextRelations` | `guarantee`, `idempotencyKey`, `retry`, `dlq` |

Without D0 a classifier sees "the system processes parcels" and picks
PostgreSQL because the sentence is about records.

### D1. PACELC per bounded context, then the system

Decided first, because it constrains every later node. The site gives four
classes ([]{#cite-2}), and that closed set is the whole palette:

| class | inside a partition | in the normal case | fits |
|---|---|---|---|
| `PA/EL` | availability over consistency | latency over consistency | feeds, catalogs, social |
| `PC/EC` | consistency over availability | consistency over latency | money, inventory, booking |
| `PA/EC` | availability over consistency | consistency over latency | read paths over a consistent store |
| `PC/EL` | consistency over availability | latency over consistency | latency-critical writes with read repair |

- decider: classifier. Four options, premises are NFR rows and the workload
  profile, which is the best case for scoring.
- per service first, then a system summary: the system class is a summary of the
  context classes, not a vote. A parcel system can be `PC/EC` for payment and
  `PA/EL` for tracking.
- gate: every bounded context has a class; the class must be one of the four; a
  `PC/EC` context cannot get an AP-only store or an async-only relation.
- lands in: `HLD.boundedContexts[].pacelc` and a `decisions[]` entry with
  `techs: []` and `objectRefs` naming the context's containers.
- evidence: an NFR row (consistency or availability metric) or a BRD constraint.

### D2. Failure profile

Derived from D1 and the NFRs, by code, not by a model: `rpoMinutes`,
`rtoMinutes`, retry budget, and the consistency model the SRS promises
(linearizable, serializable, read-your-writes, eventual, per the Jepsen chapter
[]{#cite-3}). This profile is the input to D5 and D7; it is stored once and read
everywhere.

### D3. Language and team stack

Tool first, product second. The language is not a design choice when the team
already runs one:

1. `premises.team.languages` comes from the operator;
2. the system takes that language as a rule. The palette holds one option, and
   the decision records the premise as its evidence;
3. only an explicit operator request to learn a new language opens a second
   option, and then it is a TEJ cost, not a style preference.

The LLD path root follows a table in code (`go → internal/<layer>/<module>`,
`java → src/main/java/<module>/<layer>`, `ts → src/<layer>/<module>`), so H1
disappears: the model no longer interprets a string.

### D4. Storage: mechanism first, brand second

The site's selection framework ([]{#cite-4}) separates the workload profile from
the engine: OLTP or OLAP, read/write ratio, consistency, replication, sharding,
TCO, operational risk. The chain is the same:

1. **mechanism** from D0 and D2: transactions and joins, access by key, wide rows
   with aggregation, graph closure, object bytes, full-text, time series;
2. **class**: relational, key-value/in-memory, document, wide-column, search,
   time-series, graph, vector, object;
3. **product**, filtered by budget, team stack and licence: one class, at most
   five products.

Only step 3 is a classifier, and it is cheap because the class is already fixed by
premises. The chosen product becomes a container with its `tech`, which the
existing gate already requires; a product without a container stays impossible.

Replication and sharding are decided in the same node when the premise says so
(`premises.traffic`, `premises.storage`): leader-based replication, leaderless
with `R + W > N`, or none; sharding by key with consistent hashing when a hot key
is plausible ([]{#cite-5}).

### D5. Communication

Per relation: sync or async, driven by D1 and D2. The site's communication
patterns ([]{#cite-6}) give the palette and the guarantee each option buys:

| option | guarantee | cost |
|---|---|---|
| REST | request/response, easy contracts, no streaming | chatty, latency per hop |
| gRPC | typed contract, streaming, low overhead | schema tooling, harder debugging |
| GraphQL | client-shaped reads, one round trip | query complexity limits |
| message queue | at-least-once, load leveling | consumer idempotency |
| pub/sub topic | fan-out, at-least-once | ordering per key only |
| event stream | replay, ordered per partition | operational weight |

The HLD already carries `type: sync|async`; this node adds `guarantee`,
`idempotencyKey`, `retry` and `dlq`, because an async relation without a duplicate
policy is a design that loses money on retries ([]{#cite-7}).

### D6. Architecture style

The last big choice, not the first. It reads the TEJ verdict and budget, the SRS
NFRs (availability, p99, team), and D1 to D5.

- palette: modular monolith, layered monolith, hexagonal module set, microservices,
  event-driven, cell-based.
- decider: classifier. The premises are numeric (team size, budget, SLO, number of
  contexts) and the option descriptions are stable across projects.
- gate: the style must match the container count and the relation types. Nine
  deployed services are not a modular monolith; one deployable unit is not
  microservices. This is exactly the contradiction the live ADR produced.
- evidence: `feasibility.recommendedOption`, `feasibility.premises`,
  `srs.nfr[]`.

The site's decomposition rule ([]{#cite-6}) is the tie-breaker: boundaries follow
capabilities, bounded contexts, team topology and data ownership, not the number
of nouns in the request.

### D7. Reliability and integration patterns, conditional on D4 and D5

This node is a tree, not one choice. Each branch is skipped when its premise is
absent, and each answer is a named mechanism with a guarantee ([]{#cite-8}):

| premise | question | palette | what it guarantees |
|---|---|---|---|
| two callers reach one key at once | concurrency control | optimistic version, pessimistic lock, single writer | no lost update vs no blocking |
| a call can time out | retry policy | none, retry + backoff, retry + circuit breaker | delivery under transient faults vs thundering herd |
| a write and an event must agree | transactional outbox | outbox + CDC, listen-to-yourself, none + compensation | atomic state change and publish |
| one operation spans three containers | saga | orchestration, choreography, 2PC | completion without a global lock |
| a message can fail forever | dead letter | DLQ, parking lot, none | poison isolation vs silent loss |
| a producer is faster than a consumer | backpressure | bounded queue + block, drop, degrade, scale | no unbounded memory growth |
| a lock must survive a pause | fence | lease + fencing token | stale owner cannot write |

Kafka-specific branches ([]{#cite-9}): producer acks and `min.insync.replicas`
decide whether a write survives a broker loss; consumer groups decide the
parallelism; ordering is per partition, so the partition key is a design decision,
not an afterthought; delivery is at-least-once in practice, and exactly-once is a
consumer-side effect achieved by an idempotency key or an idempotent sink.

### D8. Platform and its own checklist

The platform is a decision with three options, and each brings its own list
([]{#cite-10}):

- **Kubernetes**: one deployment per container, replica floor and ceiling, pod
  disruption budget, readiness and liveness probes backed by a real endpoint,
  resource requests and limits, autoscaling on a real metric, config and secrets
  injection, network policy between contexts, multi-region topology when the RTO
  demands it.
- **VM**: systemd unit, log rotation, disk and backup, manual rolling restart.
- **Serverless**: cold start budget, concurrency limit, payload size, state
  ownership, idempotency on retries.

The checklist is generated from the decision, so a VM design never answers a
question about pods.

### D9. Observability and success metrics

Closes the loop: p99, availability SLO, throughput, error rate, plus the business
metric of the BRD goal. It reads D1, D6 and D8, and writes `decisions[]` plus the
NFR rows that measure it.

## 3. What the classifier decides, and what it must not

| node | decider | why |
|---|---|---|
| D1 PACELC class | classifier | four options, premises are numbers |
| D4 product inside a class | classifier | closed list, budget filter |
| D6 architecture style | classifier | six stable options, numeric premises |
| D7 pattern per branch | classifier | one branch, two or three options |
| D8 platform | classifier | three options, hard operational premises |
| rejection reason that cites an FR sentence | LLM | the reason must quote the requirement |
| invariants, contracts, module paths | LLM or code | language, not selection |

Mechanics, in the order the code needs them:

1. the palette lives in the registry and in `domain` as a closed list, so the
   prompt and the gate read one source;
2. a hard filter in Go removes excluded options (budget, language, licence,
   existing tech) before the model sees them;
3. scoring is `argmax` over what is left: a zero-shot classifier or token
   log-probabilities of the local model, with `{option, confidence}` stored;
4. below a threshold the decision becomes an operator question with the same
   options, and the answer is stored with `source: operator`;
5. every decision records the author (`classifier`, `llm`, `operator`) and the
   confidence. Without that, a wrong classifier call is indistinguishable from a
   wrong model call.

## 4. Code-first plan

Each step ships alone and lands in existing layers.

1. **Premises (D0).** Add `premises` to `Feasibility` and the per-FR fields to
   `FunctionalRequirement`; ask them in the stage 01 and 02 interviews; gate them.
   No model needed. Files: `domain/form_feasibility.go`, `domain/form_srs.go`,
   `domain/form_validate.go`, `form-registry/feasibility.md`, `form-registry/srs.md`,
   `usecase/prompt.go`.
2. **Closed tech list and the path table (H1).** An enum in `domain`, a
   `tech → path root` table for the LLD, and the HLD gate on membership.
3. **PACELC per context (D1).** Fields on `BoundedContext`, the four-class palette,
   one decision per context, and the gate that refuses a class the store or the
   relation cannot provide.
4. **Architecture style (D6).** Premise: TEJ verdict plus NFR rows; gate: style
   against container count and relation types.
5. **Communication guarantees (D5) and `deps` ↔ relations (H4).** The LLD gate
   that refuses a dependency no HLD relation allows.
6. **Reliability tree (D7).** A conditional sub-chain in the HLD skeleton and one
   decision per branch that fired.
7. **Platform checklist (D8).** A `platform` decision, then the checklist of that
   platform only.
8. **Classifier adapter.** Behind the same palette and gate, starting with D1 and
   D6; measure agreement against the decisions already stored.

## 5. Anti-goals

- A classifier before a palette, a hard filter and a threshold: it inherits the
  same vagueness and hides it behind a number.
- One CAP choice for the whole system. The class is per context; the system view
  is a summary.
- Free-text technology anywhere. A product no container carries is a promise.
- A Kubernetes checklist in a VM design, or a saga in a system with one container.
- A decision without an evidence link. "Industry best practice" is not evidence.
- Classifying the reason for a rejection. The reason is a sentence about a
  requirement, and that is what the model is for.

## Sources

The taxonomy and the option palettes above follow the chapters of
[system-design.space](https://system-design.space/):

- [ ](#cite-1) Design process: requirements and estimate — `/theme/system-design-approaches/`
- [ ](#cite-2) CAP and PACELC classes PA/EL, PC/EC, PA/EC, PC/EL — `/chapter/cap-theorem/`, `/chapter/pacelc-theorem/`
- [ ](#cite-3) Consistency models under test (linearizable, serializable, eventual) — `/chapter/jepsen-consistency/`
- [ ](#cite-4) Database selection framework: OLTP/OLAP, read/write ratio, replication, sharding, TCO — `/chapter/database-selection-framework/`
- [ ](#cite-5) Leaderless replication, `R + W > N`, consistent hashing and hot keys — `/chapter/leaderless-replication/`, `/chapter/consistent-hashing-hot-keys-rebalancing/`
- [ ](#cite-6) Decomposition strategies and inter-service communication patterns — `/chapter/decomposition-strategies/`, `/chapter/inter-service-communication-patterns/`
- [ ](#cite-7) Reliable event delivery: Outbox, Inbox, CDC, ordering, at-least-once — `/chapter/reliable-event-delivery/`
- [ ](#cite-8) Saga, compensation, backpressure, resilience patterns, fencing tokens — `/chapter/saga-pattern/`, `/chapter/backpressure-flow-control/`, `/chapter/resilience-patterns/`, `/chapter/distributed-locks-leases-fencing/`
- [ ](#cite-9) Kafka: partitions, consumer groups, delivery guarantees — `/chapter/kafka-book/`
- [ ](#cite-10) Cloud Native, Kubernetes, GitOps, multi-region, serverless — `/theme/cloud-native-containerization/`
