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

> A question names the subject of the request it asks about: the guest, the receptionist,
> the card, the tariff. Write `<the role the request names>` and `<the word the request
> uses>` as the concrete words of the request, never as the words written here. A question
> that fits any other system the same way tells the operator nothing.
> A question never contains the characters `<` or `>`: they mark a placeholder of this
> template, not a question. Copy the word of the request into the sentence.

### Paragraph 1 — Business outcome

**Scope:** The purpose of the system on the market and the metric that proves it. Do not describe architecture.

> A greenfield system does not run yet: `metrics` stays empty unless the operator already measures the value. Do not ask for a metric name, unit, baseline or target. Put the success the operator names into the goal outcome.

**Questions:**

1. **Purpose:** Уточнить у оператора назначение системы на рынке.
   **Field:** `payload.goals[].outcome`.
   **Question:** What is the purpose of `<the system the request names>` for `<the role the request names>`?
2. **Purpose:** Уточнить у оператора метрику результата.
   **Field:** `payload.metrics[].name`.
   **Question:** Which metric proves the purpose?
3. **Purpose:** Уточнить у оператора единицу измерения метрики.
   **Field:** `payload.metrics[].unit`.
   **Question:** Which unit does the metric use?
4. **Purpose:** Уточнить у оператора текущее значение метрики, если оно известно.
   **Field:** `payload.metrics[].baseline`.
   **Question:** What is the current value of the metric, if it is known?
5. **Purpose:** Уточнить у оператора целевое значение метрики, если оно известно.
   **Field:** `payload.metrics[].target`.
   **Question:** What is the target value of the metric, if it is known?

**Answer:** `payload.goals[]` — `{ id, outcome: Text, metricId? }`.
**Answer:** `payload.metrics[]` — `{ id, name, unit?, baseline?, target? }`.

### Paragraph 2 — Project boundary

**Scope:** The capabilities inside and outside the project.

**Questions:**

1. **Purpose:** Уточнить у оператора возможности внутри границы проекта.
   **Field:** `payload.scope.in[]`.
   **Question:** Which capability must `<the system the request names>` have?
2. **Purpose:** Уточнить у оператора возможности вне границы проекта.
   **Field:** `payload.scope.out[]`.
   **Question:** Which capability does `<the system the request names>` exclude?
3. **Purpose:** Уточнить у оператора отложенные возможности.
   **Field:** `payload.scope.deferred[]`.
   **Question:** Which capability does `<the system the request names>` defer to a later release?

**Answer:** `payload.scope` — `{ in: Text[], out: Text[], deferred: Text[] }`.

### Paragraph 3 — Actors and tasks

**Scope:** The people or systems that use the target system. One actor entry per role.
The user flow of a role is a task of that role: the sign-in and the personal cabinet are
steps of the same person, not a technical detail of the system.

**Questions:**

1. **Purpose:** Уточнить у оператора имя действующего лица.
   **Field:** `payload.actors[].name`.
   **Question:** Which name does `<the role the request names>` have in this business?
2. **Purpose:** Уточнить у оператора задачу действующего лица.
   **Field:** `payload.actors[].tasks[]`.
   **Question:** Which task does `<that role>` do?
3. **Purpose:** Уточнить у оператора вход в систему.
   **Field:** `payload.actors[].tasks[]`.
   **Question:** How does `<that role>` sign in or register, and what does it present to prove who it is?
4. **Purpose:** Уточнить у оператора личный кабинет роли.
   **Field:** `payload.actors[].tasks[]`.
   **Question:** What does `<that role>` do in the personal cabinet after the sign-in?
5. **Purpose:** Уточнить у оператора частоту задачи.
   **Field:** `payload.actors[].frequency`.
   **Question:** How often does `<that role>` do this task?
6. **Purpose:** Уточнить у оператора бизнес-объект задачи.
   **Field:** `payload.actors[].businessObject`.
   **Question:** Which business object does `<that role>` change?

**Answer:** `payload.actors[]` — `{ id, name, tasks[], frequency, businessObject }`.
**Answer:** `tasks[0]` is the main task, `tasks[1]` the sign-in and `tasks[2]` the personal cabinet.

### Paragraph 4 — Glossary

**Scope:** The words of the business in the operator's own terms: a person, a document, an event, a device. The design stages read this before they name an entity or a container.

**Questions:**

1. **Purpose:** Уточнить у оператора значение бизнес-слова.
   **Field:** `payload.glossary[]`.
   **Question:** What does the word `<the business word the request uses>` mean in this business, in the real world?

**Answer:** `payload.glossary[]` — `Text[]`. One sentence holds the word and its meaning.

### Paragraph 5 — Constraints and assumptions

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
5. **Purpose:** Уточнить у оператора текущий порядок работы.
   **Field:** `payload.assumptions[]`.
   **Question:** How does the business do this work today, who does what now, and where does the work get lost?
6. **Purpose:** Уточнить у оператора процесс оплаты.
   **Field:** `payload.assumptions[]`.
   **Question:** Who pays for the service, when does the money move, and how does it arrive?
7. **Purpose:** Уточнить у оператора канал уведомления.
   **Field:** `payload.assumptions[]`.
   **Question:** Which events must the system report, on which channel do they arrive, and who receives them?
8. **Purpose:** Уточнить у оператора внешнюю систему или устройство.
   **Field:** `payload.assumptions[]`.
   **Question:** Which device or external system does the business use, and may the system read it?

**Answer:** `payload.constraints[]` — `Statement[]`.
**Answer:** `payload.assumptions[]` — `[Text]`.

### Paragraph 6 — Success measurement

**Scope:** The event and the time window that prove success.

> Ask these only when the operator already measures the value. A new system has no measurement yet: leave `successMetrics` empty instead of inventing a metric, a threshold or a window.

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

### Paragraph 7 — Missing data

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
7. **One entry per subject.** Ask for every role and every capability of the request
   separately: `actors[0].name`, `actors[1].name`, and so on, then
   `scope.in[0]`, `scope.in[1]`. Never merge two roles or two capabilities in one
   answer. A merged answer hides a role, and every later document loses it.
8. **Business words only.** Do not ask for an id, an internal name, a unit or a
   baseline. The analyst assigns the id, and a new system has no baseline.
9. **Every role has its own flow.** Ask for the sign-in and the cabinet of every
   role separately: `actors[0].tasks[1]`, `actors[1].tasks[1]`, and so on. A role
   without a flow is a name in a table, not an actor of the system.
10. **The money and the message.** Ask how the payment arrives and on which channel
   the notification reaches the role. The design stages turn both into a container
   only when the BRD names them.

## Gates

1. `goals` contains at least one entry.
2. `metrics` is optional: a new system has no measured value yet. A `goals[].metricId` or `successMetrics[].metricId` may be empty. A non-empty `metricId` MUST reference an ID that exists in `metrics[]`; do not invent metric IDs.
3. Each `metrics[]` entry has `id` and `name`. `unit`, `baseline`, and `target` are filled only when the operator knows them; a new system usually has no baseline.
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
