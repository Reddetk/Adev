# Form Registry — adev

Источник правды для форм, которыми обмениваются узлы eino-графа.
Агент заполняет абзацы формы. Агент не добавляет абзацы и поля.

Статус: **HLD, draft**. Go-типы ещё не написаны; имена полей отсюда — основа для кодогенерации.

## Что такое форма

Форма — шаблон из нескольких абзацев. Каждый абзац имеет:

1. `Scope` — одну предметную область.
2. Вопросы — только по этой области.
3. `Answer` — имя поля в `payload` и тип ответа.

Ответ — одно значение: enum, число, `objectId` или `Text`. Ответ-нарратив запрещён. Если данных нет, агент создаёт `QUESTION` с `Gap`, а не догадку.

## Правила реестра

1. Один `formType` = один файл.
2. Имена полей — camelCase, совпадают с JSON и Go.
3. `R` — обязательное, `O` — опциональное.
4. Свободный текст только в `Text`, `Statement`, `Scenario`. Эти поля проверяет `language-checker` (ASD-STE100 / ACE).
5. Принятая форма (`approved`) неизменяема. Исправление — новая `formVersion`.
6. Delta — тот же `formType` с `mode: "delta"`, `base` и JSON Patch. Новый тип не создаётся.
7. Новый `formType` требует обоснования: переиспользование обязательно (SysReq п.5).
8. `targetObjects` в каждой форме — то, что форма меняет или читает в **targetSystem**.
9. Любая ссылка на существующий объект targetSystem идёт через `objectId` из [реестра объектов](_objects.md). Путь без `objectId` допустим только для нового объекта.

## Пайплайны (happy path)

```
greenfield:
USER_INPUT → BRD → FEASIBILITY → SRS → HLD → ADR → LLD → TASK_PLAN
                                                          ↓
                         TEST_CONTRACT → IMPLEMENTATION_SPEC → VERDICT
                                                                  ↓ fail
                                                            REFACTOR_SPEC → VERDICT
change:
USER_INPUT → CR_BRIEF → IMPACT_ASSESSMENT
                              ├─ route A → SRS(delta) → LLD(delta) → TASK_PLAN
                              └─ route B → SRS(delta) → HLD(delta) → ADR → LLD(delta) → TASK_PLAN
                                                                          ↓
                                                          (далее общая TDD-петля)
```

`QUESTION` может быть создан на любом узле и блокирует переход до ответа оператора.

## Индекс

| formType | Stage | Owner | Файл | Следующий |
| --- | --- | --- | --- | --- |
| `USER_INPUT` | — | Operator | [user-input.md](user-input.md) | BRD / CR_BRIEF |
| `BRD` | 00 | Analyst | [brd.md](brd.md) | FEASIBILITY |
| `FEASIBILITY` | 01 | Analyst | [feasibility.md](feasibility.md) | SRS |
| `CR_BRIEF` | 00 | Analyst | [cr-brief.md](cr-brief.md) | IMPACT_ASSESSMENT |
| `IMPACT_ASSESSMENT` | 01 | Analyst | [impact-assessment.md](impact-assessment.md) | SRS |
| `SRS` | 02 | Analyst | [srs.md](srs.md) | HLD |
| `HLD` | 03, 04 | Architect | [hld.md](hld.md) | ADR / LLD |
| `ADR` | 04 | Architect | [adr.md](adr.md) | LLD |
| `LLD` | 05 | Analyst | [lld.md](lld.md) | TASK_PLAN |
| `TASK_PLAN` | 06 | Analyst | [task-plan.md](task-plan.md) | TEST_CONTRACT |
| `TEST_CONTRACT` | 07 | QA | [test-contract.md](test-contract.md) | IMPLEMENTATION_SPEC |
| `IMPLEMENTATION_SPEC` | 08 | Developer | [implementation-spec.md](implementation-spec.md) | VERDICT |
| `VERDICT` | 09 | QA | [verdict.md](verdict.md) | REFACTOR_SPEC / done |
| `REFACTOR_SPEC` | 08.R | Developer | [refactor-spec.md](refactor-spec.md) | VERDICT |
| `QUESTION` | any | Any | [question.md](question.md) | Operator |

Служебные файлы: [_envelope.md](_envelope.md), [_objects.md](_objects.md), [_specificity-report.md](_specificity-report.md).
