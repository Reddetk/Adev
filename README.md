# adev

adev turns one business request into a design dossier: a business requirements document, a
techno-economic justification, a requirements specification, a high-level design, the
architecture decisions, a low-level design, a task plan and a test contract. Ten stages, one
document each, one role agent per stage, gates on every result, and one question for every
value the model must not invent.

The rule the whole system is built around: **the model supplies meaning, the code supplies
structure, the operator supplies business decisions.** ids, references, layout keys, retries
and ordering never go through the model; the model never sees a form it cannot answer.

Setup and commands: `adev/README.md`. Coding rules: `AGENTS.md`. Form templates:
`form-registry/`.

## What it does

**A fixed route of documents.** `domain/stage.go` owns the order and the owner of each stage.
The agent never chooses the next stage, so two runs on the same input walk the same path.

```
00 BRD → 01 TEJ → 02 SRS → 03 HLD → 04 invariants + ADR → 05 LLD → 06 plan → 07 tests → 08 impl → 09 verdict
```

**An interview, not a form fill.** Stage 00 asks the operator about the business in the
operator's own words, not about the fields of the schema. The current set of questions:

- the purpose of the business, and the name of every role the request names;
- **the user flow of every role**: how the role enters the system, what it presents to prove
  who it is (`actors[i].tasks[1]`), and what it does in its personal cabinet (`tasks[2]`);
- how the money arrives, which event the notification carries and on which channel;
- **how the business runs today**, before the system: who does what on paper, and where the
  work gets lost;
- the business words the request never defines, the external devices, the project boundary,
  and the one limit that applies.

The interview repeats after every batch of answers, so a small model is never asked to invent
15 capabilities at once.

**One level of nesting per model call.** A composite document is written in two phases: a
skeleton call lists the entries, then one call per entry fills it — one per FR, one per
container, one per package, one per decision. The model sees one entry and the document it
belongs to, never the whole list.

**A distributed system, not a box of nouns.** The HLD names the infrastructure as containers
of its own: the reverse proxy / API gateway at the edge, the identity provider (Keycloak) when
the BRD has a sign-up or a sign-in, the message bus (Kafka) when a step is asynchronous, a
load balancer, a scheduler, and the external devices. Each one gets its own bounded context,
its own degradation rule, and a relation that wires it in: sync for a call that waits for the
answer, async with an `eventRef` for an event the bus carries.

**Ten architecture decisions, in dependency order.** The ADR covers architecture style,
communication, data storage, scaling, consistency (CAP/PACELC, decided after storage and
scaling), reliability, build stack, operations and security, design process and success
metrics. Every decision names the option taken and at least two rejected ones, and cites the
SRS requirement or NFR that decided it — never a form the stage does not read.

**Questions only about the business.** A gap carries a JSON pointer, one question and optional
choices. The system drops a question about an id, a reference, a type name, a layout key, a
currency or an amount: the operator decides meaning, the analyst names the mechanics. Answers
patch the stored draft in place, so a decision lands where the uncertainty was.

**A model that fails does not take the stage down.** A transient model error retries ten times
with doubling backoff (0.5 s → 5 s). A structural hole in the HLD or the LLD (no layer, no
package path, an aggregate without an invariant) gets up to three model passes before a
question goes out. A gate mismatch is stored on the draft as an indicator, and the console
shows it instead of hiding it.

**The UI is the operator console.** Three tabs: Target System (requirements, context map,
aggregates, C4 container map), Pipeline (the stage chain, live run cards, the model-call
history with prompts and responses, the stop button) and Settings (target system, workspace,
one agent config per role). The questions dashboard answers a gap without leaving the page.

**Local models are first-class.** Any OpenAI-compatible provider works; a local Ollama model
runs the whole route at no cost, which is how the prompts are tested live.

## How it works

![Who decides what in one stage run](docs/diagrams/stage-run.svg)

```
request ──► stage runner ──► context builder ──► model call(s) ──► assembler ──► gates ──► stored version
                 │                                    │                           │
                 │                                    └── one entry + one document│
                 └── operator questions ◄──────────── open gaps ◄──────────────────┘
```

Diagram source: `docs/diagrams/stage-run.html`.

- **One document per stage, one owner agent.** A stage reads its input forms, writes its output
  form. It never writes a document another stage owns.
- **The model proposes, the code assembles.** After every answer the system assigns the ids,
  restores the references the model dropped, keeps the fields the skeleton already held, and
  coerces a plain string into the `Text` record the shape requires.
- **The accepted version is the floor.** When a stage rewrites a document that already passed
  (stage 04 rewrites the HLD of stage 03), every accepted value is inherited unless the new
  answer replaces it. A model that returns an empty field cannot delete a decision.
- **Gates judge the document, not the model.** A validation failure is a stored draft plus an
  indicator; only the operator decides whether to rerun. There is no silent self-repair loop.
- **The route belongs to the registry.** A failed verdict enters 08.R; a change request takes
  the change route. The same inputs always walk the same path.

## Concurrency, and where it stops

The detail walk is a **fork/join per level of the document**:

```
level 1  ── fork: one goroutine per entry ──┐
                                           ├── results channel (one buffer slot per fork)
         ── join: wait, close, merge ───────┘
level 2  (children of an entry start after its own answer)
```

- **Fork per entry, join per level.** Each entry of a list becomes one worker;
  `detailLevel` closes the results channel only after every fork returned. Siblings run at
  once, children wait for their parent's answer.
- **One writer.** The receiver merges results one at a time; workers never touch the document,
  so no lock is needed. Thread safety comes from the shape, not from a mutex.
- **A bounded slot channel, not a bounded goroutine count.** Every model call takes a slot of
  `detailSem` (4 in-flight calls by default). The fork count follows the document, the cost
  follows the provider limit.
- **Context inheritance down the tree.** A child call carries its own entry, the path of
  parents, and the document as read-only context: the branch knows where it sits without
  retyping the document.
- **Retry is resilience, not a loop.** A transient failure backs off and retries the same
  call; it never re-plans the entry, so a retried call cannot change the shape of the answer.

The pipeline stays sequential between stages on purpose: a stage reads what the previous stage
wrote. Parallelism inside a stage is safe because the split is a table (`batchPlans`), not a
model decision.

## Patterns we use

| Pattern | Where | Why |
|---|---|---|
| Prompt chaining | every stage: read inputs → build context → call → assemble → gate → store | one narrow job per step; a failure points at one step |
| Orchestrator–workers | composite documents: skeleton, then one call per entry, forked by level | the document is longer than one context; each entry gets its own small prompt |
| Routing | `domain.NextStage` and the route table | the path is a rule, so the run is auditable |
| Human in the loop | gaps, operator requests, answers | an assumption the operator never saw is a defect with a delay |
| Guarded generation | gate-first submit, accepted-version inheritance, payload coercion | the model may be wrong; the document stays usable |

## Patterns we refuse

**Evaluator–optimizer loops.** One attempt per form. An auto-repair loop multiplies token cost
and hides the moment the model stopped being able to answer.

**ReAct.** The model never picks an action and never reads an observation back. It receives one
unit of context and returns one unit of text; code owns the order and the tools. We give up
self-correction and buy a route that cannot drift and a cost that can be estimated.

## Expansion plans

- **Make the concurrency budget a setting.** Per-provider workers and a token budget per stage,
  instead of one global slot count.
- **A clean-slate pass.** Today a rewrite inherits the previous accepted version, which keeps a
  weak design alive; add an explicit "regenerate without inheritance" run.
- **Bind ADR evidence to ids.** A decision should link the specific FR/NFR rows it serves, so a
  changed requirement marks the decisions that must be revisited.
- **Estimation as a first-class step.** Traffic, storage and growth belong in their own form
  between the SRS and the HLD, where the scaling and CAP/PACELC decisions can cite them.
- **Finish the change route.** CR brief and impact assessment exist in the registry; the loop
  back into SRS and HLD is next.
- **Close the loop to code.** LLD → implementation spec → test contract → CI, with the same
  gate-first discipline.
- **A prompt regression harness.** Golden forms and a diff of the produced documents per model,
  so a prompt edit is measured instead of felt.
- **Per-stage model choice.** A small model for skeletons and normalization, a large one for
  semantics and trade-offs.

## Where to look

```
form-registry/      form templates and gates, the source of truth for prompts
adev/internal/domain/       entities, routes, gates, payload shapes — no I/O
adev/internal/usecase/      stage runner, prompts, questions, assembly, inheritance
adev/internal/repository/   pgx adapters, OpenAI-compatible model client with retries
adev/internal/handler/http/ REST API the console uses
frontend/           operator console (React + TypeScript + Vite)
docs/               requirements, architecture, diagrams
```
