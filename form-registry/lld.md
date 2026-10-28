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
2. Which layer name applies?
3. Which package path holds the code?
4. Which module path holds the behavior?

**Answer:** `payload.containers[]` — `{ containerId, layers[] }`.
**Answer:** `payload.containers[].layers[]` — `{ name, packages[] }`.
**Answer:** `payload.containers[].layers[].packages[]` — `{ path, modules[] }`.
**Answer:** `payload.containers[].layers[].packages[].modules[]` — `{ objectId, path }`.

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
10. Each `openGaps[].pointer` names one value of `payload`, not a whole list or entry.

## Delta

`base` is the approved `LLD`. The `ops` patch changes `/payload/containers`, `/payload/contracts`, `/payload/paths`, and `/payload/eventMaps`.
Route A changes only contracts and paths inside an existing container.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `layer` | create / change | `name` is one of `adapter`, `domain`, `app`, `infra`. |
| `package` | create / change | Register the package with `status: planned`. |
| `module` | create / change | Register the module with `status: planned`. |
| `broker` | create / change | Register the topic or queue with `status: planned`. |
| `config` | read | Read the current configuration. |

## Specificity note

A layer name outside the enum is a defect. A module without `objectId` or `path` is a defect.
