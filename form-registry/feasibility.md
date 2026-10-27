# FEASIBILITY — Techno-Economic Justification (ТЭО)

| field | value |
| --- | --- |
| formType | `FEASIBILITY` |
| stage | `01` |
| owner | `analyst` |
| route | `greenfield` |
| mode | `full` |
| inputs | `BRD`, as-is `HLD` |
| next | `SRS` |

The greenfield route writes a TEJ: it prices the options against the BRD goals
and picks a verdict. A system that does not exist yet has no running baseline,
so a risk register is out of scope here: a change route with an accepted as-is
HLD can assess risks on its own stage instead.

## Template

### Paragraph 1 — Option set

**Scope:** The solution options that the analyst compares.

**Questions:**
1. Which option can satisfy the BRD goals?
2. Which option is the smallest change to the current system?
3. Which option uses only existing target-system objects?

**Answer:** `payload.options[]` — `{ id, summary: Text, reused: [objectId], created: [TargetRef] }`.

### Paragraph 2 — Cost and schedule

**Scope:** The resources that each option needs.

**Questions:**
1. Which currency does the cost use?
2. What is the cost amount?
3. How many work days does the option need?
4. What is the confidence of the estimate?

**Answer:** `payload.options[].cost` — `{ currency, amount }`.
**Answer:** `payload.options[].scheduleDays` — integer.
**Answer:** `payload.options[].confidence` — `low`, `medium`, `high`.

### Paragraph 3 — Recommendation

**Scope:** The selected option and the verdict.

**Questions:**
1. Which option does the analyst recommend?
2. Which verdict follows from the analysis?
3. Which reason supports the verdict?

**Answer:** `payload.recommendedOption` — `option.id`.
**Answer:** `payload.verdict` — `go`, `no_go`, `revise`.
**Answer:** `payload.reason` — one `Text` value.

### Paragraph 4 — BRD correction

**Scope:** The BRD scope that the study removes or changes.

**Questions:**
1. Which BRD scope item is not feasible?
2. Which replacement scope item does the analyst propose?

**Answer:** `payload.brdRef` — `Ref` to the current BRD.
**Answer:** `payload.scopeDelta[]` — `{ removed, added, reason: Text }`.

## Gates

1. `recommendedOption` exists in `options`.
2. Each `options[].reused[]` value exists in the object registry.
3. `verdict: no_go` stops the pipeline until the operator closes the pipeline or creates a `QUESTION`.
4. `verdict: revise` returns a new `BRD` version.
5. `scopeDelta` is not empty when `verdict` is `revise`.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| `container` | read | Read the as-is HLD and the object registry. |
| `doc` | read | Read the as-is HLD document. |

## Specificity note

`confidence` is an enum, not a percentage. A percentage without a method is a guess.
