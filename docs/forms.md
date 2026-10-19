# Forms

Строгая структура форм для маршрутов из `Arhitecture.md`.
Цель: агент не угадывает ни одного поля. Всё, что он читает и пишет, — типизировано и проверяемо кодом.

---

## 1. Три решения, на которых держится всё

1. **Один конверт + атомы.** Нет 14 независимых схем. Есть один `FormEnvelope` и 9 переиспользуемых атомов. Форма стадии — это конверт с конкретной композицией атомов в `payload`.
2. **Дельта — это JSON Patch, а не новый тип формы.** Delta SRS / Delta HLD / Delta LLD = та же форма с `mode: "delta"`, `base` и `ops` (RFC 6902). Новых смысловых сущностей не появляется (штраф из SysReq п.5).
3. **Проекция — это allowlist JSON Pointer'ов по роли.** Агент получает не форму целиком, а отфильтрованное поддерево. Нет доступа — поля физически нет в контексте.

Следствие: валидатор форм — обычный детерминированный код, без LLM. LLM не пишет формы произвольно, а заполняет поля, которые ему разрешены проекцией.

---

## 2. Конверт (обязателен для любой формы)

```jsonc
{
  "formId": "FRM-01J9Z8K3T4",        // ULID, неизменен
  "formType": "SRS",                // enum, см. таблицу §4
  "schemaVersion": 1,               // версия схемы, не формы
  "formVersion": 1,                 // версия этого документа, растёт при правке
  "previousFormVersion": null,      // ref | null
  "status": "draft",                // draft | valid | waiting_operator | approved | rejected | superseded
  "route": "greenfield",            // greenfield | change
  "stage": "02",                    // "00".."09" | "08.R"
  "mode": "full",                   // full | delta
  "base": null,                     // ref | null; обязателен при mode=delta
  "ops": null,                      // JSON Patch [] | null; обязателен при mode=delta
  "planId": "PLAN-01J9Z8...",       // один на всю цепочку
  "producedBy": "SysA",             // SysA | Arh | QA | Dev | DevOps | Operator
  "inputs": [                       // ровно те формы, что разрешены стадии
    { "formId": "FRM-...", "formType": "BRD", "formVersion": 2, "hash": "sha256:..." }
  ],
  "payload": { },                   // см. §4
  "payloadHash": "sha256:..."       // канонический JSON payload после применения ops
}
```

Правила:

- `formId` + `formVersion` уникальны. `payloadHash` считается после материализации дельты.
- Принятая форма (`approved`) неизменяема. Исправление = `formVersion + 1` и `previousFormVersion`.
- `inputs` — закрытый список. Если данных нет в `inputs`, агент не имеет права их «вспоминать»; он создаёт `QUESTION`.
- Свободный текст разрешён только в полях типа `Text`. Все значения `Text` прогоняются через `language-checker` (ASD-STE100), нормативные — через ACE. Имена полей и enum-значения проверяются линтером на этапе CI.

`ref`:

```jsonc
{ "formId": "FRM-...", "formType": "LLD", "formVersion": 3, "pointer": "/payload/containers/0", "hash": "sha256:..." }
```

---

## 3. Атомы (переиспользуются в payload всех форм)

| Атом | Поля | Зачем |
| --- | --- | --- |
| `Ref` | `formId, formType, formVersion, pointer?, hash` | любая ссылка; запрещает пересказ |
| `Text` | `text, lang: "STE100"\|"ACE"` | единственный носитель свободного текста |
| `Statement` | `id, text, modality: "shall"\|"should"\|"may", priority: "must"\|"high"\|"low"` | FR, NFR, инвариант, AC |
| `TargetRef` | `kind: container\|layer\|package\|module\|table\|broker\|test\|config\|pipeline\|env\|doc, path, containerId?` | привязка к физической реализации (`targetObjects.md`) |
| `Contract` | `id, moduleRef: TargetRef, kind: http\|grpc\|event\|function\|repo, inputRef, outputRef, errors: [{code, httpStatus?, when: Text}], invariants: [Statement]` | единица LLD и тестирования |
| `TestCase` | `id, contractRef, targetTest: TargetRef, kind: unit\|contract\|integration, given: [Text], when: [Text], then: [Text], path: happy\|negative, expectedErrorCode?` | TDD-контракт |
| `Task` | `id, parentId?, title: Text, owner: Role, taskType, inputs: [Ref], outputs: [TargetRef], acceptance: [Statement], dependsOn: [taskId], externalKey?` | дерево задач |
| `Decision` | `id, title: Text, context: Text, decision: Text, alternatives: [{option: Text, rejectedBecause: Text}], consequences: [Text], supersedes?: decisionId` | ADR |
| `Risk` | `id, description: Text, likelihood: 1..5, impact: 1..5, mitigation: Text, refs: [Ref]` | ТЭО, impact |
| `Gap` | `id, formRef, pointer, question: Text, options: [Text], blocking: bool` | неопределённость → оператор |

`Contract`, `TestCase`, `Task`, `Statement` — те же имена, что в `targetObjects.md`. Отдельных «форм-контрактов» не вводим.

---

## 4. Формы по маршруту

Один `formType` = одна схема payload. Дельта любого типа использует тот же тип.

| Stage | formType | Owner | Payload (поля верхнего уровня) | Потребитель |
| --- | --- | --- | --- | --- |
| — | `USER_INPUT` | Operator | `rawRequest: Text, attachments: [Ref], receivedAt` | 00 |
| 00 (green) | `BRD` | SysA | `goals, scope{in,out}, actors, constraints, assumptions, successMetrics, openGaps` | 01, 02 |
| 00 (change) | `CR_BRIEF` | SysA | `changeType: business\|devsecops\|techdebt\|audit, driver, currentBehavior, requestedBehavior, urgency, domains, evidence: [Ref]` | 01 |
| 01 (green) | `FEASIBILITY` | SysA | `options[{id,cost,schedule,risks}], recommendedOption, risks, verdict: go\|no_go\|revise, brdRef` | 02, Operator |
| 01 (change) | `IMPACT_ASSESSMENT` | SysA | `blastRadius[{target: TargetRef, changeKind, depth}], impactedForms: [Ref], nfrBaselineDelta, routing{route: A\|B, ruleId, reason: Text}, risks` | 02 |
| 02 | `SRS` | SysA | `fr: [Statement], nfr[{statement, metric, threshold, verifyMethod}], outOfScope, traceToGoals` | 03, 06, 07 |
| 03 | `HLD` | Arh | `ubiquitousLanguage[{term,definition,sourceRef}], boundedContexts[{id,name,purpose,containers}], containers[{id,purpose,boundedContext}], aggregates[{id,containerId,root,eventMaps}], contextRelations[{from,to,type: sync\|async}]` | 04 |
| 04 | `HLD` (v2) | Arh | те же поля + `aggregates[].invariants: [Statement]` + `degradationStrategy[{failure,degradation,detection}]` | 05 |
| 04 | `ADR` | Arh | `decisions: [Decision], status: proposed\|accepted\|superseded` | 05, 08 |
| 05 | `LLD` | SysA | `containers[{containerRef, layers[{name,packages[{path,modules[{path,deps}]}]}]}], contracts: [Contract], paths[{id,contractRef,kind:happy\|negative,given,when,then}], eventMaps[{event,producer,consumers,payloadRef,idempotencyKey}], adrRefs` | 06, 07, 08 |
| 06 | `TASK_PLAN` | SysA (+TL) | `tasks: [Task], roots: [taskId], criticalPath: [taskId]` | 07 |
| 07 | `TEST_CONTRACT` | QA | `taskRef, tests: [TestCase], coverage{contractsCovered, pathsCovered, uncovered: []}, ready: bool` | 08, 09 |
| 08 | `IMPLEMENTATION_SPEC` | Dev | `taskRef, testContractRef, targets[{target: TargetRef, action: create\|modify\|delete, generator: codegen\|configgen\|manualException}], steps, constraints, exception?{reason, approvalRef}` | 09 |
| 09 | `VERDICT` | QA | `taskRef, implementationRef, results[{testId,status:pass\|fail,evidence: Ref}], verdict, violations[{testId,aspect,severity,evidence}]` | 06 (reopen), 08.R |
| 08.R | `REFACTOR_SPEC` | DevOps → Dev | `verdictRef, violations, suggestedPatterns, preserveTests: true, allowedFiles: [TargetRef]` | 09 |
| любая | `QUESTION` | любой | `gaps: [Gap], waitingSince` | Operator |

Замечания по маршруту:

- `BRD-clean` — не новый тип, а `BRD` `formVersion: 2` после удаления отклонённого scope. `FEASIBILITY.brdRef` указывает на неё.
- `Semantic Analysis` (stage 03) и `Invariant Analysis` (stage 04) — это проверки, а не формы. Их результат лежит в `HLD.ubiquitousLanguage` и `HLD.aggregates[].invariants`. Отдельную форму не заводим.
- `Impact Assessment` уже содержит `routing`, поэтому «Routing Decision» отдельной формой не является. Route A/B — поле, а не новый артефакт.
- `Task Plan` в change-маршруте — тот же `TASK_PLAN`; каждая задача обязана ссылаться на `base`-задачу или на `Delta LLD` (поле `inputs`).
- В `Arhitecture.md` stage 08.R помечен `DevOps`, хотя меняет код. Предлагаю owner `Dev`, а `DevOps` — только если меняются `pipeline`/`env`. Это надо подтвердить.

---

## 5. Инварианты структуры

Проверяются кодом, не агентом:

1. `formVersion = previousFormVersion.version + 1`, иначе отказ.
2. `mode: "delta"` требует `base` того же `formType` и `status: approved`; результат патча проходит ту же схему.
3. `payload` не содержит `Text` вне атомов `Text`/`Statement`/`Gap`. Поиск по схеме: все строковые поля либо enum, либо `Text`.
4. Каждый `Ref` разрешается в существующую форму; `hash` совпадает. Битая ссылка = `status: rejected`.
5. `inputs` каждого stage ⊆ разрешённого списка (см. §4). Лишний вход = отказ, а не «дополнительный контекст».
6. `producedBy` соответствует owner стадии.
7. `status: approved` ставит только гейт: схема + линтер + трассировка. Для `BRD`, `HLD`, `ADR` дополнительно `Operator`.
8. `QUESTION` с `blocking: true` переводит план в `waiting_operator`; никакие другие стадии не стартуют.
9. Дерево задач: ровно один `parentId` или `roots`; циклов нет; у листа есть `acceptance` и после stage 07 — минимум один `TestCase`.
10. Трассировка (жёсткие гейты):
    - `FR` → хотя бы один `Contract` или `Task`;
    - каждый `Contract` → хотя бы один `TestCase` (`kind: contract`);
    - каждый `path` из `LLD` (`happy` и `negative`) → `TestCase`;
    - каждый `Statement` из `acceptance` → `TestCase`;
    - `VERDICT: pass` возможен только если все `TestCase` задачи имеют `status: pass`.
11. `IMPLEMENTATION_SPEC`: `generator: manualException` допустим только с непустым `exception.approvalRef` на `Operator`.

Именно п.10 делает «TDD» не лозунгом, а блокирующей проверкой.

---

## 6. Проекции (что видит агент)

Проекция = `{formType → [JSON Pointer]}` для роли. Пример для `LLD`:

| Pointer | SysA | Arh | Dev | QA | DevOps |
| --- | --- | --- | --- | --- | --- |
| `/payload/contracts` | R | R | R | R | – |
| `/payload/contracts/*/errors` | R | R | R | R | – |
| `/payload/containers/*/layers/*/packages/*` | – | – | R | – | – |
| `/payload/containers/*/layers/*/packages/*/modules/*/deps` | R | – | R | – | R |
| `/payload/paths` | R | R | – | R | – |
| `/payload/adrRefs` | R | R | R | R | – |
| `/*/payloadHash`, `/*/inputs` | R | R | R | R | R |

Пример выдачи разработчику на stage 08:

```jsonc
{
  "task": { /* Task целиком */ },
  "tests": { /* TEST_CONTRACT.tests */ },
  "contracts": [ /* только Contract этой задачи */ ],
  "adr": [ { "id": "ADR-012", "decision": "..." } ],   // без alternatives/consequences
  "allowedTargets": [ /* TargetRef из IMPLEMENTATION_SPEC */ ]
}
```

Разработчик не получает BRD, HLD, `impactedForms`, `blastRadius`, `alternatives` из ADR. Архитектор не получает `module.code` и `layers`. Это и есть «hides data from agents» из SysReq.

Правило для схем: проекция не может расширять доступ. Если поле не указано в матрице `targetObjects.md`, оно недоступно.

---

## 7. Примеры

### BRD (сокращённо)

```jsonc
{
  "formId": "FRM-01J9Z8K3T4", "formType": "BRD", "schemaVersion": 1,
  "formVersion": 1, "previousFormVersion": null, "status": "valid",
  "route": "greenfield", "stage": "00", "mode": "full", "base": null, "ops": null,
  "planId": "PLAN-01J9Z8K000", "producedBy": "SysA",
  "inputs": [{ "formId": "FRM-01J9Z8K000", "formType": "USER_INPUT", "formVersion": 1, "hash": "sha256:aa" }],
  "payload": {
    "goals": [{ "id": "GOAL-1", "text": "The system shall build software from a user request.", "metric": "cycle_time_hours" }],
    "scope": { "in": ["planning", "tdd"], "out": ["hosting"] },
    "actors": ["operator", "developer", "architect"],
    "constraints": [{ "id": "C-1", "text": "The system shall use local models.", "modality": "shall" }],
    "assumptions": [],
    "successMetrics": [{ "id": "M-1", "name": "first_pass_tests", "target": ">= 0.8" }],
    "openGaps": []
  },
  "payloadHash": "sha256:bb"
}
```

`USER_INPUT` — единственный тип вне стадий: сырой запрос оператора, сохраняется как форма, чтобы `inputs` был полным и проверяемым.

### TEST_CONTRACT (сокращённо)

```jsonc
{
  "formId": "FRM-...", "formType": "TEST_CONTRACT", "formVersion": 1,
  "status": "valid", "route": "greenfield", "stage": "07", "mode": "full",
  "planId": "PLAN-...", "producedBy": "QA",
  "inputs": [{ "formId": "FRM-LLD", "formType": "LLD", "formVersion": 3, "hash": "sha256:cc" }],
  "payload": {
    "taskRef": { "taskId": "TASK-12", "planId": "PLAN-..." },
    "tests": [{
      "id": "TC-12-1", "contractRef": "CT-44", "kind": "contract", "path": "negative",
      "targetTest": { "kind": "test", "path": "internal/auth/token_validator_test.go" },
      "given": [{ "text": "The token is expired.", "lang": "STE100" }],
      "when":  [{ "text": "The validator reads the token.", "lang": "STE100" }],
      "then":  [{ "text": "The validator returns error code TOKEN_EXPIRED.", "lang": "STE100" }],
      "expectedErrorCode": "TOKEN_EXPIRED"
    }],
    "coverage": { "contractsCovered": ["CT-44"], "pathsCovered": ["CT-44/negative"], "uncovered": [] },
    "ready": true
  }
}
```

---

## 8. Что должен делать валидатор (минимальный набор)

Один бинарь `formcheck`, без LLM, вход — файл формы:

1. JSON Schema (2020-12) по `formType`.
2. `apply(base, ops)` при `mode: delta`, затем схема.
3. `hash` всех `inputs` и `payloadHash`.
4. `language-checker` по каждому `Text`.
5. Гейты §5, включая трассировку.
6. Проекция по роли: `formcheck -role Dev -form lld.json` → отфильтрованный JSON. Агент получает только выход этой команды.

Только после `payloadHash` и `approved` форма попадает в `inputs` следующей стадии.

---

## 9. Открытые вопросы (нужно решение до схем)

1. **Who is TL** в stage 06 Route A — отдельная роль или `SysA`? В матрице `targetObjects.md` роли TL нет.
2. **Owner 08.R** — `Dev` или `DevOps`? Сейчас в `Arhitecture.md` `DevOps`, но меняется код.
3. **`USER_INPUT` и `OPERATOR_COMMAND`** (approve/reject/abort/answer) — это формы или события вне формы? Предлагаю форму для первого и событие для второго, чтобы прерывание не ждало валидации.
4. **Где живёт `planId`-состояние** (текущий stage, ожидания, блокировки): в отдельном `PLAN`-документе или выводится из графа форм. Предлагаю выводить, чтобы не было второго источника правды.

---

## 10. Порядок внедрения

1. `FormEnvelope` + 9 атомов + `BRD` → `SRS` → `TASK_PLAN` (зелёный маршрут до задач).
2. `LLD` → `TEST_CONTRACT` → `IMPLEMENTATION_SPEC` → `VERDICT` (TDD-петля, ядро системы).
3. Дельта-режим (JSON Patch) и `IMPACT_ASSESSMENT`.
4. Проекции по ролям.
5. `QUESTION`/оператор и гейты прерывания.

Так первый рабочий цикл даёт запуск задачи, а не полный документооборот.
