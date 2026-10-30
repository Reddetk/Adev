# ADR — Architecture Decision Record

| field | value |
| --- | --- |
| formType | `ADR` |
| stage | `04` |
| owner | `architect` |
| route | `greenfield`, `change` |
| mode | `full`, `delta` |
| inputs | `HLD`, `SRS` |
| next | `LLD` |

## Template

### Paragraph 1 — Decision identity

**Scope:** The ID, the status, and the previous decision.

**Questions:**
1. Which ID identifies the decision?
2. Which status does the decision have?
3. Which earlier decision does this decision replace?

**Answer:** `payload.decisions[]` — `{ id, title: Text, status, supersedes? }`.

### Paragraph 2 — Context

**Scope:** The facts that force a decision.

**Questions:**
1. Which SRS requirement, SRS NFR or HLD constraint forces the decision?
2. Which target-system object does the decision affect?

> The ADR reads the HLD and the SRS. It does not read the BRD: a rejection that cites a BRD
> goal or a BRD constraint cites a form this stage never sees.

**Answer:** `payload.decisions[].context` — one `Text` value.
**Answer:** `payload.decisions[].objectRefs[]` — `objectId` values.

### Paragraph 3 — Decision

**Scope:** The selected option in one statement.

**Questions:**
1. Which option does the architect select?
2. Which rule makes the option correct?

**Answer:** `payload.decisions[].decision` — one `Text` value.

### Paragraph 4 — Alternatives

**Scope:** The options that the architect rejected.

**Questions:**
1. Which alternative did the architect reject?
2. Which fact makes the alternative wrong?
3. Which BRD goal, BRD constraint or SRS NFR is that fact?

**Answer:** `payload.decisions[].alternatives[]` — `{ option: Text, rejectedBecause: Text }`.
**Answer:** at least two rejected options per decision. A rejection cites evidence the input forms hold, never a rule the BRD does not state.

### Paragraph 5 — Consequences

**Scope:** The result of the decision for the target system.

**Questions:**
1. Which quality improves?
2. Which quality becomes worse?
3. Which object changes because of the decision?

**Answer:** `payload.decisions[].consequences[]` — `[Text]`.

### Paragraph 6 — Links

**Scope:** The forms that the decision affects.

**Questions:**
1. Which `HLD` aggregate does the decision constrain?
2. Which `LLD` contract does the decision constrain?

**Answer:** `payload.decisions[].links[]` — `Ref[]`.

## Decision areas

One decision per area, in this order. A later decision rests on an earlier one, so the
set reads as a chain: the style decides the communication, the communication and the
storage decide the scaling, and the storage and the scaling decide CAP/PACELC.

| # | area | options to weigh |
| --- | --- | --- |
| 1 | architecture style | layered, modular monolith, hexagonal, microservices, event-driven, serverless, cell-based |
| 2 | communication | REST, gRPC, GraphQL, message queue, pub/sub topic, event stream, webhook, WebSocket, server-sent event, long polling |
| 3 | data storage | relational, key-value, document, wide-column, graph, object storage |
| 4 | scaling | load balancing, caching layer, replication, sharding, autoscaling |
| 5 | consistency | CAP/PACELC trade-off, strong versus eventual |
| 6 | reliability | rate limit, circuit breaker, retry with idempotency |
| 7 | build stack | database, cache, message bus, gateway, search, runtime |
| 8 | operations and security | observability, load and chaos test, delivery, OAuth 2.0/OIDC, TLS, WAF, secrets vault |
| 9 | design process | the requirement and the NFR the choice serves, the traffic and storage estimate |
| 10 | success metrics | p99 latency, availability, throughput, error rate |

## Gates

1. Each decision has at least one rejected alternative.
2. `supersedes` points to an existing decision.
3. `status: accepted` requires an `HLD` version of at least v2.
4. `status: superseded` requires a replacement decision.
5. Each `objectRefs[]` value exists in the object registry.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `doc` | create / change | Write `ADR-<nnn>.md` to the target-system repository. |

## Specificity note

`status` is an enum. A decision without a rejected alternative is a preference, not a decision.
