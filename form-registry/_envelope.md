# Form Envelope and Atoms

Every form uses this envelope. Do not repeat these fields in a form file.

## Envelope

| field | type | req | rule |
| --- | --- | --- | --- |
| `formId` | string | R | `FRM-<ULID>`. This value never changes. |
| `formType` | enum | R | See `README.md`. |
| `schemaVersion` | int | R | Version of the schema. |
| `formVersion` | int | R | Version of this document. Minimum 1. |
| `previousFormVersion` | `Ref?` | O | Required when `formVersion > 1`. |
| `status` | enum | R | `draft`, `valid`, `waiting_operator`, `approved`, `rejected`, `superseded`. |
| `route` | enum | R | `greenfield`, `change`. |
| `stage` | string | R | `00` to `09`, `08.R`, or `—`. |
| `mode` | enum | R | `full`, `delta`. |
| `base` | `Ref?` | O | Required when `mode` is `delta`. |
| `ops` | JSON Patch | O | RFC 6902. Required when `mode` is `delta`. |
| `pipelineId` | string | R | `PIPE-<ULID>`. One value for the full pipeline. |
| `targetSystemId` | string | R | `TGT-<ULID>`. Selects one object registry. |
| `producedBy` | enum | R | `operator`, `analyst`, `architect`, `qa`, `dev`, `devops`. |
| `inputs` | `Ref[]` | R | Closed list of stage inputs. |
| `payload` | object | R | Answers for the form template. |
| `payloadHash` | string | R | `sha256:<hex>` of the canonical payload after the `ops` patch. |

## Atoms

| atom | fields |
| --- | --- |
| `Ref` | `formId, formType, formVersion, pointer?, hash` |
| `Text` | `text, lang: "STE100" \| "ACE"` |
| `Statement` | `id, text, modality: "shall" \| "should" \| "may", priority: "must" \| "high" \| "low"` |
| `Scenario` | `given: [Text], when: [Text], then: [Text]` |
| `TargetRef` | `objectId?, kind, path?, containerId?, new?`. `objectId` is mandatory for an existing object. `new: true` requires `path` and `kind`. The registry assigns the final `objectId` at approval. |
| `Contract` | `id, moduleRef: TargetRef, kind: "http" \| "grpc" \| "event" \| "function" \| "repo", inputRef, outputRef, errors: [{code, httpStatus?, when: Text}], invariants: [Statement]` |
| `TestCase` | `id, contractRef, targetTest: TargetRef, kind: "unit" \| "contract" \| "integration", scenario: Scenario, path: "happy" \| "negative", expectedErrorCode?` |
| `Task` | `id, parentId?, title: Text, owner, taskType, inputs: [Ref], outputs: [TargetRef], acceptance: [Statement], dependsOn: [taskId], externalKey?` |
| `Decision` | `id, title: Text, context: Text, decision: Text, alternatives: [{option: Text, rejectedBecause: Text}], consequences: [Text], supersedes?` |
| `Risk` | `id, description: Text, likelihood: 1 to 5, impact: 1 to 5, mitigation: Text, refs: [Ref]` |
| `Gap` | `id, formRef, pointer, question: Text, options: [Text], blocking: bool` |

A form file defines the atoms that only that form uses.

## Answer rules

1. Answer each question with one value, one enum value, or one `objectId`.
2. Do not write a narrative answer.
3. Use `Text` only when no enum value and no `objectId` exists.
4. Every `Text` value must pass `language-checker`.
5. Every reference uses `formId` and `pointer`. Do not copy data from the source form.
6. Each paragraph limits its answers to its own scope. An answer outside the scope is a form defect.
7. If the data is absent, create a `Gap`. Do not guess.

## Projection

An agent receives an allowlist of JSON Pointers for its role.
A field that is absent from `targetObjects.md` and from the role projection is absent from the agent context.
