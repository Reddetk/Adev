# Отчёт о предметности

Что было нечётким в первой версии реестра и что изменено.
Формулировки вопросов — ASD-STE100 (короткие предложения, активный залог, один вопрос = одно данное).

## Сквозные пробелы (были во всех формах)

| Пробел | Где проявлялся | Исправление |
| --- | --- | --- |
| Нет идентичности объекта | `path`, `string[]`, «модуль вообще» | `objectId` из [_objects.md](_objects.md); путь только для нового объекта |
| Нет адресата ответа | `Text` без критерия | каждый абзац: `Scope` → вопросы → `Answer` с типом |
| Нет единиц измерения | NFR, cost, timeout | `unit`, `timeoutMs`, `currency` обязательны |
| Нет точки и окна измерения | `verifyMethod` без места | `measurementPoint`, `window` в NFR |
| Нет запрета на нарратив | свободные строки | правило §Answer в `_envelope.md` |
| Нет привязки к абзацу | payload-поля вразброс | абзац → ключ payload |
| Нет проверки «данных нет» | агент писал прозу | `Gap` + `QUESTION`, ответ оператора = новая версия формы |

## По формам

| Форма | Место без предметности | Что сделано |
| --- | --- | --- |
| `USER_INPUT` | `source` — строка | enum `operator/api/issue`; `locale` enum |
| `BRD` | `actors: string[]` | `actor + task + frequency + businessObject` |
| `BRD` | `successMetrics` без окна | `threshold + window` |
| `BRD` | `goals` без базовой линии | `baseline` и `target` в `metrics` |
| `FEASIBILITY` | «риски» без шкалы | `likelihood 1..5`, `impact 1..5`, `mitigation` |
| `FEASIBILITY` | `cost` без валюты | `currency + amount` |
| `FEASIBILITY` | оценка без уверенности | enum `confidence`, запрещён процент без метода |
| `CR_BRIEF` | `currentBehavior` — проза | `objectId + input + output + error` |
| `CR_BRIEF` | `domains` — строка | `boundedContext.id` из реестра |
| `IMPACT_ASSESSMENT` | «радиус» без объекта | `objectId + changeKind + depth` |
| `IMPACT_ASSESSMENT` | `routing` без правила | `ruleId` из rulebook, проверяется кодом |
| `IMPACT_ASSESSMENT` | NFR delta без единиц | `current/required/unit/verifyMethod` |
| `SRS` | FR без актора и триггера | `actor + trigger + response + stateChange` |
| `SRS` | ошибки не описаны | `errors[] {code, condition, limit}` |
| `SRS` | NFR без точки измерения | `measurementPoint + window` |
| `HLD` | `tech` — свободный текст | ID из approved technology list |
| `HLD` | aggregate без реализации | `moduleRef` обязателен |
| `HLD` | деградация без таймаута | `timeoutMs` |
| `ADR` | решение без альтернативы | минимум одна `rejectedBecause` |
| `ADR` | нет связи с объектами | `objectRefs[]` |
| `LLD` | `Layer.name` — свободный | goca-слои `domain/usecase/repository/handler/di/middleware`, atomic `token/atom/molecule/organism/template/page`; старые `adapter/app/infra` валидны |
| `LLD` | контракт без ошибок | `errors[]` обязателен для каждого контракта |
| `LLD` | путь без кода ошибки | `expectedErrorCode` для negative |
| `LLD` | событие без идемпотентности | `idempotencyKey + ordering` |
| `TASK_PLAN` | «работает корректно» | `acceptance` с value + unit + test |
| `TASK_PLAN` | нет тестовой ссылки | `acceptanceRefs[]` (fr/path) |
| `TASK_PLAN` | роль `TL` | удалена |
| `TEST_CONTRACT` | «возвращает ошибку» | `expectedErrorCode` |
| `TEST_CONTRACT` | покрытие без списка | `contractsCovered/pathsCovered/uncovered` |
| `IMPLEMENTATION_SPEC` | `steps` без цели | `targetObjectId + action` в каждом шаге |
| `IMPLEMENTATION_SPEC` | код от LLM без контроля | `manualException` + `approvalRef` |
| `VERDICT` | `skipped` не определён | `skipped` = `fail` без approval |
| `VERDICT` | нет доказательства | `runId` + `evidence` у каждого результата |
| `REFACTOR_SPEC` | можно менять тесты | `preserveTests` всегда `true` |
| `REFACTOR_SPEC` | нет списка файлов | `allowedFiles` без `kind: test` |
| `QUESTION` | «нужно ли это?» | поле + факт + варианты ответа |
| `QUESTION` | ответ правит вопрос | ответ создаёт новую версию формы-источника |

## Что осталось решить

1. **Rulebook маршрутизации** (`rules/routing.yaml`) и **approved technology list** — где хранятся и кто меняет.
2. **Владелец реестра объектов** — кто выдаёт `objectId`: отдельный сервис adev или детерминированный генератор от `targetSystemId + kind + counter`.
3. **`AS_IS_HLD`** — отдельная форма или `HLD` с `mode: as-is`.
4. **Jira `externalKey`** — обязателен ли для `route: change`.
5. **Кто создаёт файл теста** — QA на stage 07 или dev на stage 08. Сейчас `TEST_CONTRACT` регистрирует `test` как `planned`, файл создаёт `IMPLEMENTATION_SPEC` через `codegen`.
