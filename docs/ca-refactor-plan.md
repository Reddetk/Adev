# План приведения adev к Clean Architecture (4 плоских слоя)

**Статус: выполнен.** Этапы 1–4 применены, `go build`, `go vet`, `go test ./...` зелёные.
Удалены `internal/{forms,registry,pipeline,settings,constants}` и три дублирующие сущности
(`FormRecord`, `FormRef`, `ObjectEntry`); миграция `005_registry_and_form_dedup`.
Осталось из плана: этап 5 (HTTP для форм, pgx-стор объектов, stage 00 с eino).

Решение: строгое следование каркасу goca — `domain` / `usecase` / `repository` / `handler`.

План убирает самодельные пакеты вне слоёв и сводит дублирующиеся сущности к одной.
Поведение HTTP не меняется до этапа 5.

---

## 1. Правила слоёв, которые вводим

| Слой | Можно | Нельзя |
| --- | --- | --- |
| `domain` | сущности, value objects, enum'ы, чистые функции и правила переходов | I/O, `context.Context`, SQL, HTTP, `os`, импорт других слоёв, внешние форматы |
| `usecase` | сервисы, DTO, **порты** (интерфейсы хранилищ и внешних инструментов, которые слой потребляет) | SQL, HTTP, импорт `repository` |
| `repository` | pgx-адаптеры, реализующие порты из `usecase` | бизнес-правила |
| `handler` | http/cli/worker адаптеры, маппинг ошибок в статусы | бизнес-логика, SQL |
| `di` | корень композиции: собирает порты и сервисы | логика |

Четыре следствия:

1. **Порты объявляет `usecase`.** Сейчас `usecase` импортирует `repository` — зависимость наружу.
2. **В `domain` нет `context.Context`.** Сейчас `ValidateEnvelope(ctx, …)` и `Store` с ctx — это I/O в домене.
3. **`messages` и `constants` — не слои.** `constants` удаляется (0 использований). `messages` остаётся только для уцелевшего CRUD, в новом коде не используется.
4. **`middleware` и `di` остаются** — инфраструктура адаптера и корень композиции.

---

## 2. Решения, принятые в этом плане

| # | Решение | Почему |
| --- | --- | --- |
| D1 | Порты — в `usecase`; `repository` только реализует | направление зависимостей внутрь |
| D2 | `domain` не знает про `context.Context` | домен перестаёт зависеть от транспорта и времени жизни запроса |
| D3 | Разрешение ссылок и языковая проверка — в `usecase`, в домен передаются уже готовые факты | убирает порты `Resolver` и `TextChecker` из домена |
| D4 | `ApplyDelta` (JSON Patch) — в `usecase` | это материализация версии, а не правило предметной области; внешний формат не попадает в домен |
| D5 | `CanonicalJSON`/`HashPayload` — в `domain` | hash определяет тождество формы |
| D6 | `Projection` и `BRDProjection` — в `domain` | это правило видимости, чистые данные |
| D7 | `forms.Task` (атом) не создаётся; расширяется существующий `domain.Task` | иначе два `Task` |
| D8 | `registry.FormRef` удаляется в пользу `domain.Ref` | два типа одной ссылки |
| D9 | `Gap` остаётся value object в payload, `OperatorRequest` — запись в БД; маппинг 1:1 | разные роли: значение против записи |
| D10 | `pipelineId` (код) побеждает `planId` (`docs/forms.md`) | код уже работает |
| D11 | `FormService`, `SettingsService`, `RegistryService`, `PipelineService` собираются в `di`, а не в `main.go` | корень композиции один |

---

## 3. Детальная карта: `internal/forms` (2670 строк вместе с тестами)

Самый важный пакет: он смешивает домен, порты, внешний формат (JSON Patch) и SQL.

### 3.1 Что переезжает куда

| Файл | Строк | Целевой файл | Что именно | Изменения |
| --- | --- | --- | --- | --- |
| `envelope.go` | 143 | `domain/role.go` | `Role`, константы ролей, `Role.Valid()` | вынести из файла |
| | | `domain/form.go` | `FormType`+константы, `Status`+константы, `Route`, `Mode`, `Ref`, `Envelope`, `Form`, `SetPayload`, `BindPayload`, `ComputeHash` | без изменений кода |
| `atoms.go` | 63 | `domain/form_atoms.go` | `Lang`, `Text`, `Modality`, `Priority`, `Statement`, `Gap` | `TextChecker` и `Resolver` **удаляются** из домена (D3) |
| `user_input.go` | 37 | `domain/form_user_input.go` | `UserInput`, `Source`, `Locale`, `ParseReceivedAt` | без изменений |
| `brd.go` | 85 | `domain/form_brd.go` | `BRD`, `Goal`, `Metric`, `Scope`, `Actor`, `Frequency`, `Success`, `Texts()` | без изменений |
| `hash.go` | 34 | `domain/form_hash.go` | `CanonicalJSON`, `HashPayload` | без изменений (D5) |
| `projection.go` | 162 | `domain/form_projection.go` | `Projection`, `Project`, `parsePointer`, `matchPointer`, `mergeValue`, `BRDProjection` | без изменений (D6) |
| `patch.go` | 28 | `usecase/form_delta.go` | `ApplyDelta` | пакет `usecase`, зависимость json-patch остаётся здесь (D4) |
| `validate.go` | 246 | `domain/form_validate.go` | `ValidationError`, `invalid`, `knownFormTypes`, `knownStages`, `knownStatus`, `knownRoute`, `knownMode`, `knownRole`, `containsWord` | чистая валидация конверта |
| | | `domain/form_status.go` | `CanChangeStatus`, `allowedStatusChanges` | переезд из `store.go` |
| | | `domain/form_gates.go` | новый `ValidatePayload(f *Form) error` + `ValidateUserInput(in UserInput) error` + `ValidateBRD(brd BRD, status Status) error` | диспетчер гейтов по `formType`; **без** `TextChecker` |
| | | `usecase/form_language.go` | `checkTexts`, порт `TextChecker` | языковая проверка — внешний инструмент |
| `store.go` | 67 | `domain/errors.go` | `ErrFormNotFound`, `ErrVersionExists`, `ErrMissingPrevious`, `ErrInvalidStatusChange` | добавить к существующим |
| | | `usecase/form_store.go` | `FormStore` (интерфейс, те же 5 методов) | порт (D1) |
| | | `usecase/form_service.go` | `FormService` + приватный `resolveRefs` | `StoreResolver` **удаляется**, логика внутри сервиса |
| `store_memory.go` | 161 | `repository/forms_store_inmemory_test.go` | `inMemoryFormsStore`, `cloneForm`, `formKey` | тестовый двойник порта, рядом с адаптером |
| `store_postgres.go` | 245 | `repository/forms_store.go` | `FormsStore`, `NewFormsStore`, `OpenFormsStore`, `Save/Get/Latest/ByPipeline/SetStatus`, `exists`, `scanForm`, `optionalJSON`, `nullableJSON` | пакет `repository`, тип переименован `PostgresStore` → `FormsStore` |
| `forms_test.go` | 271 | см. §3.3 | | разбить |
| `store_test.go` | 181 | `repository/forms_store_test.go` | `runStoreSuite`, `buildForm`, `TestMemoryStore` | suite против обоих адаптеров |
| `store_postgres_test.go` | 78 | `repository/forms_store_postgres_test.go` | `testDatabaseURL`, `TestPostgresStore`, `TestPostgresStoreIsAppendOnly` | гейт по `ADEV_TEST_DATABASE_URL` сохраняется |

### 3.2 Как меняются сигнатуры

```go
// было (I/O и порты в домене)
func (f *Form) ValidateEnvelope(ctx context.Context, checker TextChecker, resolver Resolver) error
func ValidateBRD(brd BRD, status Status, checker TextChecker) error

// станет (чистый домен)
type ResolvedRef struct {
    Ref         Ref
    PayloadHash string
}
func (f *Form) ValidateEnvelope(resolved []ResolvedRef) error
func ValidatePayload(f *Form) error
func ValidateBRD(brd BRD, status Status) error
func ValidateUserInput(in UserInput) error
func CanChangeStatus(from, to Status) bool
```

```go
// usecase: порты и сервис
type FormStore interface {
    Save(ctx context.Context, form *domain.Form) error
    Get(ctx context.Context, formID string, version int) (*domain.Form, error)
    Latest(ctx context.Context, formID string) (*domain.Form, error)
    ByPipeline(ctx context.Context, pipelineID string, formType domain.FormType) ([]*domain.Form, error)
    SetStatus(ctx context.Context, formID string, version int, status domain.Status) error
}
type TextChecker interface{ Check(domain.Text) error }

type FormService struct {
    store   FormStore
    checker TextChecker // nil → языковая проверка выключена
}
func (s *FormService) Submit(ctx context.Context, form *domain.Form) error
func (s *FormService) Materialize(ctx context.Context, base *domain.Form, ops json.RawMessage) (*domain.Form, error)
func (s *FormService) SetStatus(ctx context.Context, formID string, version int, status domain.Status) error
```

`Submit` — единственная точка входа записи: `resolveRefs` (через `FormStore`) → `domain.ValidateEnvelope` → языковая проверка `form.Texts()` → `domain.ValidatePayload` → `FormStore.Save`.

### 3.3 Как разбиваются тесты

| Тест | Куда |
| --- | --- |
| `TestHashIsOrderIndependent` | `domain/form_hash_test.go` |
| `TestEnvelopeAcceptsValidForm`, `…RejectsDeltaWithoutBase`, `…RejectsHashMismatch`, `…RejectsVersionWithoutPrevious` | `domain/form_test.go` (сигнатура без resolver: список `ResolvedRef`) |
| `TestBRDRejects*` (4 теста), `validBRD`, `ptr` | `domain/form_brd_test.go` |
| `TestProjection*` (3 теста), `newBRDForm` | `domain/form_projection_test.go` |
| `TestApplyDelta`, `TestApplyDeltaRejectsBrokenPatch` | `usecase/form_delta_test.go` |
| `TestResolverIsCalledForInputs`, `recordingResolver` | `usecase/form_service_test.go` (заменить на фейковый `FormStore`) |
| `runStoreSuite`, `TestMemoryStore`, `buildForm` | `repository/forms_store_test.go` |
| `TestPostgresStore*` | `repository/forms_store_postgres_test.go` |

Итог: `domain` получает 5 тестовых файлов и ноль I/O; `usecase` — 2; `repository` — 2.

---

## 4. Детальная карта: `internal/settings` (513 строк)

| Файл | Строк | Куда | Что |
| --- | --- | --- | --- |
| `settings.go` | 165 | `domain/agent_config.go` | `Provider`+константы, `AgentConfig`, `AgentView`, `Validate()`, `Masked()` |
| | | `domain/errors.go` | `ErrUnknownProvider`, `ErrAPIKeyRequired`, `ErrWorkspaceRequired`, `ErrNoDatabase` |
| | | `usecase/settings_service.go` | `SettingsService` (`SetWorkspace`, `Workspace`, `SetAgent`, `Agents`, `Agent`) |
| | | `usecase/settings_store.go` | порт `AgentConfigStore` |
| | | `domain/role.go` | зависимость `forms.Role` → `domain.Role` |
| `store_postgres.go` | 138 | `repository/agent_config_store.go` | `AgentConfigStore` pgx (`PostgresAgentConfigStore`) |
| `store_memory.go` | 94 | `repository/agent_config_store_inmemory_test.go` | двойник порта |
| `settings_test.go` | 116 | `domain/agent_config_test.go` | валидация провайдера, ключа, temperature, `Masked` |
| | | `usecase/settings_service_test.go` | round-trip workspace и агентов на двойнике |

Сигнатуры: `Service.SetAgent(ctx, cfg)` → `SettingsService.SetAgent(ctx, cfg domain.AgentConfig)`; всё остальное без изменений.

---

## 5. Детальная карта: `internal/registry` (365 строк)

| Файл | Строк | Куда | Что |
| --- | --- | --- | --- |
| `object.go` | 72 | `domain/object.go` | `Kind`+константы, `Status`, `Object`, `FormRef` — **удалить** (D8, вместо него `domain.Ref`) |
| `service.go` | 102 | `usecase/registry_service.go` + `usecase/object_store.go` | `RegistryService` (`Propose/Activate/Remove/Get/List`) и порт `ObjectStore` |
| | | `domain/object.go` | `CanChange` + `allowedTransitions` (переезд из `service.go`) |
| `memory.go` | 63 | `repository/object_store_inmemory_test.go` | двойник порта |
| `registry_test.go` | 128 | `domain/object_test.go` (переходы) + `usecase/registry_service_test.go` (выдача id, список, ошибки) | разбить |

Postgres-стор реестра (`repository/object_store.go`) пишется на этапе 5, когда появляется стадия 03. До этого реестр живёт с тестовым двойником — как и сейчас, только по правилам слоёв.

---

## 6. Детальная карта: `internal/pipeline` (290 строк)

| Файл | Строк | Куда | Что |
| --- | --- | --- | --- |
| `stages.go` | 83 | `domain/stage.go` | `Stage`, `greenfieldStages`, `Stages`, `StageByID`, `Roles`, `cloneStages` |
| `readiness.go` | 88 | `domain/readiness.go` | `Readiness`, `Check` (чистая политика) |
| | | `usecase/pipeline_service.go` | `Plan`, `PipelineService` (зависит от `SettingsService`) |
| `pipeline_test.go` | 119 | `domain/stage_test.go` (маршрут, владельцы, `StageByID`; `Check` переносится сюда) + `usecase/pipeline_service_test.go` (`Plan` на двойнике настроек) | разбить |

---

## 7. Правки вне переезжаемых пакетов

| Файл | Правка |
| --- | --- |
| `handler/http/settings_handler.go` | импорты: `settings`→`usecase`, `pipeline`→`usecase`, `forms`→`domain`; в аннотациях swagger `settings.AgentView`→`domain.AgentView`, `pipeline.Plan`→`usecase.Plan` |
| `handler/http/routes.go` | удалить `SetupFormRecordRoutes`, `SetupFormRefRoutes`, `SetupObjectEntryRoutes` |
| `cmd/server/main.go` | удалить 3 блока `Setup*Routes` и дублирующий блок `router.HandleFunc` для 3 сущностей; сборку новых сервисов убрать из `main` в `di` |
| `di/container.go` | минус 3 репозитория/usecase/handler и их геттеры; плюс `FormService`, `SettingsService`, `RegistryService`, `PipelineService` и их сторы/геттеры |
| `repository/interfaces.go` | минус `FormRecordRepository`, `FormRefRepository`, `ObjectEntryRepository` |
| `messages/messages.go`, `responses.go` | минус блоки FormRecord, FormRef, ObjectEntry |
| `domain/errors.go` | минус `ErrInvalid{FormRecord,FormRef,ObjectEntry}*`; плюс сентинелы из §3.1, §4 |
| `mocks/` | удалить `mock_{formrecord,formref,objectentry}_{handler,repository,usecase}.go` и `mocks/examples/*` (9+3 файла) |

---

## 8. Что удаляется

| Удаляется | Файлов | Причина |
| --- | --- | --- |
| `internal/constants` | 1 | 0 использований |
| `domain/{formrecord,formref,objectentry}.go` + `_seeds` + `_test` | 9 | заменяются `domain.Form` и `domain.Object` |
| `usecase/{formrecord,formref,objectentry}_{service,usecase}.go` + блоки в `dto.go` | 6 | сущности уходят |
| `repository/postgres_{formrecord,formref,objectentry}_repository.go` | 3 | то же |
| `handler/http/{formrecord,formref,objectentry}_handler.go` | 3 | то же |
| `mocks/mock_{formrecord,formref,objectentry}_*` + примеры | 12 | то же |

Остаются 7 сущностей: `TargetSystem`, `Pipeline`, `Run`, `Task`, `OperatorRequest`, `ModelCall`, `AuditEvent`.

---

## 9. Миграция БД (005)

```sql
DROP TABLE IF EXISTS form_records;   -- дубль к forms
DROP TABLE IF EXISTS form_refs;      -- ссылки внутри forms.inputs
DROP TABLE IF EXISTS object_entries; -- пересоздаётся под domain.Object

CREATE TABLE object_entries (
    object_id        TEXT PRIMARY KEY,
    target_system_id BIGINT NOT NULL,
    kind             TEXT NOT NULL,
    name             TEXT NOT NULL,
    path             TEXT NOT NULL DEFAULT '',
    container_id     TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL,
    fields           JSONB NOT NULL DEFAULT '{}'::jsonb,
    introduced_by    JSONB NOT NULL,
    changed_by       JSONB NOT NULL DEFAULT '[]'::jsonb
);
CREATE INDEX object_entries_target_idx ON object_entries (target_system_id, kind, status);
```

Перед применением проверить, что три таблицы пусты:
`SELECT count(*) FROM form_records, form_refs, object_entries;`

---

## 10. Порядок работ

Каждый шаг заканчивается зелёным `make fmt && make vet && make test` и `make build`.

### Этап 0 — решения (приняты, §2)

### Этап 1 — удалить мёртвое и дубли
1. Удалить `internal/constants`.
2. Удалить 3 сущности по §8 и их обвязку.
3. Миграция 005 (§9).
4. Обновить `di`, `main.go`, `routes.go`, `repository/interfaces.go`, `messages`, `mocks`.
5. Перегенерировать swagger: `make api-docs`.
   **Готово, когда:** сборка и тесты зелёные; `/api/v1/formrecords`, `/formrefs`, `/objectentrys` → 404; `/swagger/index.html` → 200; число путей в `docs/swagger.json` уменьшилось с 25 до 19 (уходят 6 путей: 3 коллекции + 3 `{id}`).

### Этап 2 — перенос в слои, по одной сущности

| Шаг | Порядок внутри шага | Проверка |
| --- | --- | --- |
| 2.1 `pipeline` → `domain`+`usecase` | сначала (зависит только от `forms.Role`) | `Plan` отдаёт те же 7 стадий и тот же `readiness` |
| 2.2 `forms` → `domain`+`usecase`+`repository` | `domain/role.go` → `domain/form*.go` → `usecase` → `repository`; тесты по §3.3 | `POST/GET/PUT` settings-эндпоинтов не изменились; `TestPostgresStore` на тестовой БД проходит |
| 2.3 `settings` → `domain`+`usecase`+`repository` | после `forms` (нужен `domain.Role`) | те же HTTP-коды: 200, 404, 422, 503 |
| 2.4 `registry` → `domain`+`usecase` | независимо | тесты реестра на двойнике |
| 2.5 `di` | собрать 4 сервиса в контейнере | `main.go` больше не знает про сторы |

Порядок 2.1 → 2.2 обязателен: `settings` и `pipeline` импортируют `forms.Role`.

### Этап 3 — чистота оставшегося CRUD (рекомендую)
Перенести 7 интерфейсов из `repository/interfaces.go` в `usecase`, `repository` оставить только с реализациями; обновить `di` и `mocks`. После этого конвенция портов одна.

### Этап 4 — документация
Удалить `docs/forms.md`; обновить `implementation-plan.md` (пути пакетов) и `AGENTS.md` (слои, запрет самодельных пакетов, `constants` удалён).

### Этап 5 — функциональность
HTTP для форм (версии, дельты, статусы) → Postgres-стор объектов → stage 00 с агентом-аналитиком (eino + `modelgw` из `agent_configs`).

---

## 11. Что сознательно не делаем

- Не вводим DI-библиотеку: `di` остаётся ручным конструктором.
- Не делаем generic-репозиторий: 7 CRUD-адаптеров остаются как есть.
- Не генерируем Go-типы из `form-registry/*.md`: markdown остаётся спецификацией, типы пишутся руками.
- Не переносим `middleware` и `docs/` (swagger) — это не слои.
- Не трогаем `docs/Arhitecture.md` и `docs/SysReq.md`.

---

## 12. Оценка и риск

| Этап | Файлов | Оценка | Риск |
| --- | --- | --- | --- |
| 1 | ~34 удалить, ~10 правок, 1 миграция | 1–2 ч | низкий: всё удаляемое — сгенерированный параллельный код |
| 2.1 | 3 → 4 | 20 мин | низкий |
| 2.2 (`forms`) | 14 → 20 | 2–3 ч | средний: самая большая развязка, ловится компилятором |
| 2.3 (`settings`) | 4 → 6 | 40 мин | низкий |
| 2.4 (`registry`) | 4 → 5 | 30 мин | низкий |
| 2.5 (`di`) | 2 | 30 мин | низкий |
| 3 | 3 + mocks | 1–2 ч | низкий, механический |
| 4 | 3 | 30 мин | нет |

Данных не теряем: `form_records`, `form_refs`, `object_entries` в dev пусты (проверить перед миграцией 005).
Главный риск — размер diff на 2.2; поэтому внутри шага идём `domain` → `usecase` → `repository`, компилируя после каждого переезда.
