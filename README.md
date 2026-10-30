# adev

adev turns one business request into a design dossier: business requirements, a techno-economic
justification, a requirements specification, a high-level design, the architecture decisions, a
low-level design, a task plan and a test contract. Ten stages, one document each, one role agent
per stage, gates on every result, and a question for every value the model must not invent.

The model supplies meaning, the code supplies structure, the operator supplies business
decisions. Ids, references, layout keys, retries and ordering stay out of the model: it never
sees a field it cannot answer.

Setup and commands: `adev/README.md`. Coding rules: `AGENTS.md`. Form templates:
`form-registry/`.

## What it does

### A fixed route of documents

`domain/stage.go` owns the order and the owner of each stage. An agent never picks the next
stage, so the same input walks the same path.

```
00 BRD → 01 TEJ → 02 SRS → 03 HLD → 05 LLD → 06 plan → 07 tests → 08 impl → 09 verdict
```

### An interview about the business

Stage 00 asks about the operator's work, not about the fields of the schema. Today it asks:

- the purpose of the business and the name of every role in the request;
- the user flow of every role: how it enters the system, what it presents to prove who it is
  (`actors[i].tasks[1]`), and what it does in its cabinet (`tasks[2]`);
- how the money arrives, which event a notification carries and on which channel;
- how the business runs today, before the system, and where the work gets lost;
- the business words the request never defines, the external devices, the project boundary,
  and the one limit that applies.

The interview repeats after each batch of answers, so a small model answers one topic at a
time instead of inventing fifteen capabilities in one call.

### One level of nesting per model call

A composite document is written in two phases: a skeleton call lists the entries, then one call
per entry fills it, one per FR, one per container, one per package, one per decision. Each call
sees a single entry and the document it belongs to.

### Infrastructure as containers

The HLD gives the reverse proxy or API gateway, the identity provider (Keycloak), the message
bus (Kafka), the load balancer, the scheduler and the external devices a container of their own.
Each one carries a bounded context, a degradation rule and a relation: sync for a call that
waits, async with an `eventRef` for an event on the bus.

### Decisions live in the design

A new system has no decision history to record, so the choices that shape it sit inside the
HLD: storage product, message bus, consistency, failure handling, edge, identity provider. Each
decision names the option taken, at least one rejected option and why, the containers it fixes,
and the products it chooses. A product must be the tech of a container of the design, so a
database or a cache the system needs becomes a container of its own.

The ADR form stays for the change route, where "why this and not that" explains a move of a
system that already runs.

### Questions about the business only

A gap holds a JSON pointer, one question and optional choices. The system drops a question about
an id, a reference, a type name, a layout key, a currency or an amount: the operator decides
meaning, the analyst names the mechanics. An answer patches the stored draft in place.

### A failed model call does not fail the stage

A transient error retries ten times with doubling backoff, 0.5 s to 5 s. A structural hole in
the HLD or the LLD gets up to three model passes before a question goes out. A gate mismatch
stays on the draft as an indicator, and the console shows it.

### The operator console

Three tabs: Target System (requirements, context map, aggregates, C4 container map), Pipeline
(stage chain, live runs, the prompt and the response of every model call, stop) and Settings
(target system, workspace, one agent config per role). A question is answered on the same page.

### Any OpenAI-compatible provider

A local Ollama model runs the whole route at no cost, which is how the prompts get tested
against a live model.

## How it works

![Who decides what in one stage run](docs/diagrams/stage-run.svg)

```
request → stage runner → context builder → model call(s) → assembler → gates → stored version
                │                                │                       │
                │                                └── one entry + document│
                └── operator questions ←──────── open gaps ←──────────────┘
```

Diagram source: `docs/diagrams/stage-run.html`.

- One document per stage, one owner agent. A stage reads its input forms and writes its output
  form; it never writes a document another stage owns.
- The model proposes, the code assembles. After each answer the system assigns ids, restores the
  references the model dropped, keeps the fields the skeleton already held, and coerces a plain
  string into the `Text` record the shape requires.
- The accepted version is the floor. A stage that rewrites a design it already published inherits every accepted
  value unless the new answer replaces it. A model that returns an empty field
  cannot delete a decision.
- Gates judge the document. A validation failure is a stored draft plus an indicator; the
  operator decides whether to rerun. There is no self-repair loop.
- The route belongs to the registry. A failed verdict enters 08.R; a change request takes the
  change route.

## Concurrency

The detail walk is a fork/join per level of the document:

```
level 1  ── fork: one goroutine per entry ──┐
                                           ├── results channel, one slot per fork
         ── join: wait, close, merge ───────┘
level 2  (children of an entry start after its own answer)
```

- Siblings run at once; a child waits for its parent's answer.
- Workers never touch the document. The receiver merges one result at a time, so the document
  has a single writer and needs no mutex.
- Every model call takes a slot of `detailSem` (4 in flight by default). The fork count follows
  the document, the spend follows the provider limit.
- A child call carries its own entry, the path of its parents and the document as read-only
  context, so the branch knows where it sits without retyping the document.
- A retry repeats the same call with a backoff and never re-plans the entry, so the shape of the
  answer stays the same.

Stages stay sequential because a stage reads what the previous stage wrote. Inside a stage the
split comes from `batchPlans`, a table, so every run splits the same way.

## Patterns we use

| Pattern | Where | Why |
|---|---|---|
| Prompt chaining | every stage: read inputs, build context, call, assemble, gate, store | one narrow job per step; a failure points at one step |
| Orchestrator–workers | composite documents: skeleton, then one call per entry, forked by level | the document is longer than one context; each entry gets its own small prompt |
| Routing | `domain.NextStage` and the route table | the path is a rule, so a run is auditable |
| Human in the loop | gaps, operator requests, answers | an assumption the operator never saw is a defect with a delay |
| Guarded generation | gate-first submit, accepted-version inheritance, payload coercion | the document stays usable when the model is wrong |

## Patterns we refuse

Evaluator–optimizer loops. One attempt per form. An auto-repair loop multiplies token cost and
hides the moment the model stopped being able to answer.

ReAct. The model never picks an action and never reads an observation back. It receives one unit
of context and returns one unit of text; the code owns the order and the tools. That trades
self-correction for a fixed route and a cost you can estimate.

## What's next

- Per-provider worker limit and a token budget per stage, instead of one global slot count.
- A regenerate pass without inheritance. Today a rewrite keeps the previous accepted design
  alive, which is how a weak HLD survives a better prompt.
- Decisions inside the HLD link the FR and NFR rows they serve, so a changed requirement marks
  the choices to revisit.
- Traffic, storage and growth in their own form between SRS and HLD, where the scaling and CAP
  decisions can cite them.
- The change route wired back into SRS and HLD; the CR brief and the impact assessment already
  exist in the registry.
- LLD → implementation spec → test contract → CI.
- A prompt regression harness: golden forms and a diff of the produced documents per model.
- A model per stage: a small one for skeletons and normalization, a large one for semantics and
  trade-offs.

## Where to look

```
form-registry/      form templates and gates, the source of truth for prompts
adev/internal/domain/       entities, routes, gates, payload shapes, no I/O
adev/internal/usecase/      stage runner, prompts, questions, assembly, inheritance
adev/internal/repository/   pgx adapters, OpenAI-compatible model client with retries
adev/internal/handler/http/ REST API the console uses
frontend/           operator console (React + TypeScript + Vite)
docs/               requirements, architecture, diagrams
```
