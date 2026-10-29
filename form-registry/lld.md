# LLD — Low-Level Design

| field | value |
| --- | --- |
| formType | `LLD` |
| stage | `05` |
| owner | `analyst` |
| route | `greenfield`, `change` |
| mode | `full`, `delta` |
| inputs | `HLD` (v2), `ADR`, object registry |
| next | `TASK_PLAN` |

## Template

### Paragraph 1 — Container layout

**Scope:** The physical structure inside one container.

**Questions:**
1. Which `objectId` identifies the container?
2. Which layer of the container tech holds the code?
3. Which package path holds the code?
4. Which module path holds the behavior?

**Answer:** `payload.containers[]` - `{ containerId, layers[] }`.
**Answer:** `payload.containers[].layers[]` - `{ name, packages[] }`.
**Answer:** `payload.containers[].layers[].packages[]` - `{ path, modules[] }`.
**Answer:** `payload.containers[].layers[].packages[].modules[]` - `{ objectId, path }`.

The model designs the layout. Do not ask the operator for a layer, a package path or a module path.

- A layer holds only `name` and `packages`. A package holds only `path` and `modules`.
  Never put a container inside a layer or a layer inside a package.
- Every package has a non-empty `path`. A module `path` extends its package path.
  Modules are optional; a package may hold no module.
- Every HLD aggregate belongs to the `domain` layer of its container: put the
  aggregate `moduleRef` in a package under `domain`.
- A Go container uses the goca layers: `domain` for aggregates and rules,
  `usecase` for services and DTO, `repository` for store adapters, `handler` for
  HTTP, CLI and worker adapters, `di` for the composition root, `middleware` for
  transport middleware. Omit a layer the container does not need.
- A UI container uses the atomic layers in one direction: `token` defines style,
  `atom` is one control, `molecule` groups atoms, `organism` groups molecules,
  `template` lays out organisms, `page` binds a template to a route. A lower
  layer never imports a higher one.
- A package path follows the container tech: `internal/<layer>/<package>` for Go,
  `src/<layer>/<feature>` for a web UI, `lib/<layer>/<feature>` for Flutter.

### Paragraph 2 — Module dependency

**Scope:** One allowed dependency between two modules.

**Questions:**
1. Which module needs the dependency?
2. Which module provides the dependency?
3. Which direction does the call take?
4. Is the call synchronous or asynchronous?

**Answer:** `payload.containers[].layers[].packages[].modules[].deps[]` — `{ providerObjectId, direction, type }`.

### Paragraph 3 — Contract

**Scope:** One interface that a module exposes.

**Questions:**
1. Which contract ID identifies the interface?
2. Which module owns the contract?
3. Which contract kind applies?
4. Which input type does the contract accept?
5. Which output type does the contract return?

**Answer:** `payload.contracts[]` — `Contract`.

### Paragraph 4 — Error catalog

**Scope:** The errors that one contract can return.

**Questions:**
1. Which contract returns the error?
2. Which error code identifies the error?
3. Which HTTP status maps to the error?
4. Which condition causes the error?
5. Can the caller retry the operation?

**Answer:** `payload.contracts[].errors[]` — `{ code, httpStatus?, condition, retryable }`.

### Paragraph 5 — Happy path

**Scope:** One successful call through one contract.

**Questions:**
1. Which `contract.id` does the path use?
2. Which state exists before the call?
3. Which request does the caller send?
4. Which response must the caller receive?

**Answer:** `payload.paths[]` — `{ id, contractRef, kind: "happy", given, when, then }`.

### Paragraph 6 — Negative path

**Scope:** One failed call through one contract.

**Questions:**
1. Which `contract.id` does the path use?
2. Which invalid state or input exists before the call?
3. Which request does the caller send?
4. Which `errors[].code` must appear?

**Answer:** `payload.paths[]` — `{ id, contractRef, kind: "negative", given, when, then, expectedErrorCode }`.

### Paragraph 7 — Event map

**Scope:** One event and its transport.

**Questions:**
1. Which event name identifies the message?
2. Which module publishes the event?
3. Which modules consume the event?
4. Which payload contract describes the event?
5. Which key prevents duplicate processing?
6. Which delivery order does the consumer require?

**Answer:** `payload.eventMaps[]` — `{ event, producerObjectId, consumerObjectIds, payloadRef, idempotencyKey, ordering }`.

### Paragraph 8 — Decision link

**Scope:** The ADR that constrains this design.

**Questions:**
1. Which `decision.id` constrains the container layout?
2. Which `decision.id` constrains a contract?

**Answer:** `payload.adrRefs[]` — `Ref[]`.

### Paragraph 9 — Open gap

**Scope:** One value the context does not give.

**Questions:**
1. Which field of the layout is missing?
2. Which question does the operator answer?
3. Which options apply when the answer is a choice?

**Answer:** `payload.openGaps[]` — `{ id, pointer, question, options, blocking }`.

`pointer` is absolute and names one value of `payload`, for example `/containers/0/layers/0/name`.
A gap that names a whole list or a whole entry is a defect: the next run of the stage fills the structure, the operator answers one value.

## Generation order

The LLD grows as a tree, one level per pass. The first answer is the outline: the
top-level list only, with each entry's id and short fields, and every child list
empty. Then each entry gets its own pass, and that pass returns only its child list.

1. `containers[]` - the top level, one entry per HLD container.
2. `containers[].layers[]` - one pass per container.
3. `layers[].packages[]` - one pass per layer.
4. `packages[].modules[]` - one pass per package.
5. `modules[].deps[]` - one pass per module.
6. `contracts[]` is a second top level; `contracts[].paths[]` is one pass per contract.

A pass never fills the child list of its children. An empty child list is a request
for the next pass, not an open gap. This keeps each answer small and lets the gates
judge one level at a time.

## Gates

1. Each `containerId` exists in the `HLD` and the object registry.
2. Each `contract.moduleRef.objectId` exists in the container layout.
3. Each contract has at least one happy path and one negative path.
4. Each negative path has an `expectedErrorCode` from its contract.
5. At least one contract invariant covers each aggregate invariant.
6. Each event map has an `idempotencyKey` and an `ordering` value.
7. Each `module.objectId` is new or belongs to the current container.
8. A delta does not delete a contract that a current test uses, unless an `ADR` allows it.
9. Each container has at least one layer; each layer has at least one package.
10. Each package has a non-empty `path`; a layer holds only `name` and `packages`, a package only `path` and `modules`.
11. Each HLD aggregate `moduleRef` appears in the `domain` layer of its container.
12. Each `openGaps[].pointer` names one value of `payload`, not a whole list or entry.

## Delta

`base` is the approved `LLD`. The `ops` patch changes `/payload/containers`, `/payload/contracts`, `/payload/paths`, and `/payload/eventMaps`.
Route A changes only contracts and paths inside an existing container.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `layer` | create / change | `name` is a goca layer of a Go container or an atomic layer of a UI container. `adapter`, `app` and `infra` pass only on an `LLD` approved before them. |
| `package` | create / change | Register the package with `status: planned`. |
| `module` | create / change | Register the module with `status: planned`. |
| `broker` | create / change | Register the topic or queue with `status: planned`. |
| `config` | read | Read the current configuration. |

## Specificity note

A layer name outside the goca and atomic sets is a defect. A package without `path` is a defect.
A module without `objectId` or `path` is a defect. A layer or a package that
contains a whole container is a defect.
