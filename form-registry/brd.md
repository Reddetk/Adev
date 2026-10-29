# BRD — Business Requirements Document

| field | value |
| --- | --- |
| formType | `BRD` |
| stage | `00` |
| owner | `analyst` |
| route | `greenfield` |
| mode | `full` |
| inputs | `USER_INPUT` |
| next | `FEASIBILITY` |

## Template

Each question has a **Purpose** and one **Field**. The analyst asks the operator the
**Question** and writes the answer into the field.

### Paragraph 1 — Business outcome

**Scope:** The purpose of the system on the market and the metric that proves it. Do not describe architecture.

**Questions:**

1. **Purpose:** Уточнить у оператора назначение системы на рынке.
   **Field:** `payload.goals[].outcome`.
   **Question:** Which purpose does the system have on the market?
2. **Purpose:** Уточнить у оператора метрику результата.
   **Field:** `payload.metrics[].name`.
   **Question:** Which metric proves the purpose?
3. **Purpose:** Уточнить у оператора единицу измерения метрики.
   **Field:** `payload.metrics[].unit`.
   **Question:** Which unit does the metric use?
4. **Purpose:** Уточнить у оператора текущее значение метрики.
   **Field:** `payload.metrics[].baseline`.
   **Question:** What is the current value of the metric?
5. **Purpose:** Уточнить у оператора целевое значение метрики.
   **Field:** `payload.metrics[].target`.
   **Question:** What is the target value of the metric?

**Answer:** `payload.goals[]` — `{ id, outcome: Text, metricId }`.
**Answer:** `payload.metrics[]` — `{ id, name, unit, baseline, target }`.

### Paragraph 2 — Project boundary

**Scope:** The capabilities inside and outside the project.

**Questions:**

1. **Purpose:** Уточнить у оператора возможности внутри границы проекта.
   **Field:** `payload.scope.in[]`.
   **Question:** Which capability does the project include?
2. **Purpose:** Уточнить у оператора возможности вне границы проекта.
   **Field:** `payload.scope.out[]`.
   **Question:** Which capability does the project exclude?
3. **Purpose:** Уточнить у оператора отложенные возможности.
   **Field:** `payload.scope.deferred[]`.
   **Question:** Which capability does the project defer to a later release?

**Answer:** `payload.scope` — `{ in: Text[], out: Text[], deferred: Text[] }`.

### Paragraph 3 — Actors and tasks

**Scope:** The people or systems that use the target system.

**Questions:**

1. **Purpose:** Уточнить у оператора имя действующего лица.
   **Field:** `payload.actors[].name`.
   **Question:** Which name identifies the actor?
2. **Purpose:** Уточнить у оператора задачу действующего лица.
   **Field:** `payload.actors[].tasks[]`.
   **Question:** Which task does the actor do?
3. **Purpose:** Уточнить у оператора частоту задачи.
   **Field:** `payload.actors[].frequency`.
   **Question:** How often does the actor do the task?
4. **Purpose:** Уточнить у оператора бизнес-объект задачи.
   **Field:** `payload.actors[].businessObject`.
   **Question:** Which business object does the actor change?

**Answer:** `payload.actors[]` — `{ id, name, tasks[], frequency, businessObject }`.

### Paragraph 4 — Constraints and assumptions

**Scope:** Limits that the solution must obey.

**Questions:**

1. **Purpose:** Уточнить у оператора закон или регламент.
   **Field:** `payload.constraints[]`.
   **Question:** Which law or regulation limits the solution?
2. **Purpose:** Уточнить у оператора бюджет.
   **Field:** `payload.constraints[]`.
   **Question:** Which budget limit applies?
3. **Purpose:** Уточнить у оператора срок.
   **Field:** `payload.constraints[]`.
   **Question:** Which deadline applies?
4. **Purpose:** Уточнить у оператора условие успеха проекта.
   **Field:** `payload.assumptions[]`.
   **Question:** Which condition must be true for the project to succeed?

**Answer:** `payload.constraints[]` — `Statement[]`.
**Answer:** `payload.assumptions[]` — `[Text]`.

### Paragraph 5 — Success measurement

**Scope:** The event and the time window that prove success.

**Questions:**

1. **Purpose:** Уточнить у оператора метрику успеха.
   **Field:** `payload.successMetrics[].metricId`.
   **Question:** Which metric proves the goal?
2. **Purpose:** Уточнить у оператора пороговое значение успеха.
   **Field:** `payload.successMetrics[].threshold`.
   **Question:** Which value marks success?
3. **Purpose:** Уточнить у оператора окно измерения.
   **Field:** `payload.successMetrics[].window`.
   **Question:** Over which window does the operator measure the value?

**Answer:** `payload.successMetrics[]` — `{ metricId, threshold, window }`.

### Paragraph 6 — Missing data

**Scope:** Data that the operator must supply.

**Questions:**

1. **Purpose:** Уточнить у оператора отсутствующее данное.
   **Field:** `payload.openGaps[]`.
   **Question:** Which field of this form has no data?
2. **Purpose:** Уточнить у оператора блокирующую роль пробела.
   **Field:** `payload.openGaps[].blocking`.
   **Question:** Does the gap block the next stage?

**Answer:** `payload.openGaps[]` — `Gap[]`.

## Question rules

1. **One question per run.** The analyst sends one question to the operator and stops.
   The analyst writes the answer into the field before it sends the next question.
   One run writes one field, not one paragraph.
2. **One datum per question.** Each question requests one value.
   The value goes into one **Field**. A question without a field is a defect. Remove it.
3. **Purpose.** Each question has one purpose. The purpose starts with
   `Уточнить у оператора` and names the datum. The purpose is the deduplication key.
4. **No logical duplication.** Before the analyst writes a question, it compares the
   purpose and the question text with every earlier question in this file.
   Two questions must not request the same datum. A repeated datum is a defect:
   keep the earlier question and remove the later one.
5. **One question for a scalar field.** A scalar field (`metrics[].unit`,
   `successMetrics[].window`) takes one question. A second question for the same
   scalar field is a defect.
6. **ASD-STE100.** Each question is one sentence, in the active voice, of 20 words or
   fewer. One question asks one thing. No semicolons. No phrasal verbs.

## Gates

1. `goals` contains at least one entry.
2. Each `goals[].metricId` exists in `metrics`. `goals[].metricId` and `successMetrics[].metricId` MUST reference an ID that exists in `metrics[]`. If unsure, use the first metric ID from `metrics[]`. Do not invent new metric IDs for goals/successMetrics.
3. Each `metrics[]` entry has `unit`, `baseline`, and `target`.
4. No field describes a container, module, or code.
5. The analyst closes all `blocking` gaps before `status: approved`.
6. `BRD-clean` is a new `formVersion` after the analyst removes rejected scope.
7. Each question has one **Field** and one **Purpose**.
8. Each **Purpose** starts with `Уточнить у оператора`.
9. No two questions request the same datum.
10. The analyst sends one question in one run.

## Object use

| object kind | relation | rule |
| --- | --- | --- |
| none | none | The BRD describes business data only. |

## Specificity note

`frequency` is an enum: `day`, `week`, `month`, `quarter`, `event`.
`businessObject` is a business noun, not a target-system `objectId`.
