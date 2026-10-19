# USER_INPUT

| field | value |
| --- | --- |
| formType | `USER_INPUT` |
| stage | `—` |
| owner | `operator` |
| route | `greenfield`, `change` |
| mode | `full` |
| inputs | none |
| next | `BRD` or `CR_BRIEF` |

## Template

### Paragraph 1 — Request source

**Scope:** Who sent the request and when.

**Questions:**
1. Which source sent the request?
2. When did the source send the request?
3. Which locale does the request use?

**Answer:** `payload.source`, `payload.receivedAt`, `payload.locale`.

### Paragraph 2 — Requested result

**Scope:** The result that the target system must give. Do not describe the solution.

**Questions:**
1. Which result must the target system give?
2. Which current behavior is wrong or absent?
3. Which user receives the result?

**Answer:** `payload.rawRequest` — one `Text` value.

### Paragraph 3 — Evidence

**Scope:** Files and records that support the request.

**Questions:**
1. Which file supports the request?
2. Which record ID identifies the file?

**Answer:** `payload.attachments[]` — `Ref[]`.

## Gates

1. `rawRequest.text` is not empty.
2. `receivedAt` is an RFC3339 value.
3. Each `attachments[]` reference resolves to an existing file.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| none | none | The request does not touch the target system yet. |

## Specificity note

`source` and `locale` are enums. A free-text source name is a defect.
