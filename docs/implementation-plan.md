# adev on eino — gap analysis and implementation plan

Дата: 2026-10-19. Статус: план, не реализация.
Основа: `SysReq.md`, `Arhitecture.md`, `form-registry/`, `targetObjects.md`.

## 0. Текущее состояние

В репозитории есть только документация и реестр форм. Нет:

- `go.mod` и Go-кода;
- схем форм, Go-типов и валидатора;
- сервиса, БД, очереди, API;
- реализации реестра объектов;
- языкового чекера (папка `language-checker` в текущем дереве отсутствует).

То есть eino не «дополняет готовую систему» — он будет фундаментом, вокруг которого нужно построить весь runtime adev.

## 1. Вывод

**eino закрывает слой оркестрации, агентов, инструментов и HITL. Он не закрывает продукт.**

| Слой | Кто делает |
| --- | --- |
| Графы стадий, ветвления, параллельность | eino `compose` |
| Агенты, ReAct, вызов инструментов, события | eino `adk` |
| Пауза/возобновление, approval | eino interrupt/resume |
| Модели (Ollama, OpenAI-совместимые, OpenRouter, Qwen) | eino + eino-ext |
| Формы, схемы, версии, дельты, проекции | **adev** |
| Реестр объектов targetSystem | **adev** |
| Пайплайн-оркестратор, очередь, lease, retry | **adev** |
| Контекст и RBAC по полям | **adev** |
| Codegen tools, sandbox, тесты, verdict | **adev** |
| Квоты, бюджет, routing моделей | **adev** |
| Operator UI, аудит, Jira/Git | **adev** |

Главное правило: **маршрут пайплайна принадлежит оркестратору и графу, не агенту.** Агент заполняет форму одной стадии. Он не выбирает следующую стадию.

## 2. Что берём из eino

| Возможность | API / пакет | Где используем |
| --- | --- | --- |
| Типизированный граф | `compose.Graph[I,O]`, `AddLambdaNode`, `AddChatModelNode`, `AddBranch`, `AddEdge` | Граф одной стадии и TDD-подграф |
| Workflow с field mapping | `compose.Workflow` | Параллельные проверки формы (схема, язык, ссылки) |
| Агент с инструментами | `adk.NewChatModelAgent`, `adk.NewRunner`, `AgentEvent` | Роль-агент внутри узла стадии |
| Инструменты | `tool.BaseTool`, `tool/utils.InferTool`, `schema.ToolInfo` | `submit_form`, чтение реестра, codegen, тесты |
| Граф как инструмент | `graphtool.NewInvokableGraphTool` | TDD-подграф как инструмент задачи |
| Пауза/возобновление | `compose.Interrupt`, `StatefulInterrupt`, `ResumeWithData`, `ExtractInterruptInfo` | `QUESTION`, approval, `manualException` |
| Чекпоинты | `compose.CheckpointStore`, `WithCheckPointStore`, `WithCheckPointID` | Восстановление после падения воркера |
| Failover моделей | `adk.ModelFailoverConfig`, `ModelRetryConfig` | Переключение local → cloud → slow backup |
| Стриминг | `Stream`, `AsyncIterator` | Прогресс стадии в UI |
| Callbacks | `callbacks.Handler`, `OnStart/OnEnd/OnError` | OTel, метрики, стоимость |
| Middleware | `adk` filesystem, summarization, skill, plantask | Долгие задачи, файлы, сжатие контекста |
| Модели | `eino-ext/components/model/{ollama,openai,openrouter,qwen,deepseek}` | Локальные и облачные модели |
| Инструменты экосистемы | `eino-ext/components/tool/{commandline,mcp,httprequest}` | Git/CI/MCP, но только через allowlist |

Пин версии: **eino v0.9.x + совместимый eino-ext**. API v0.7+ для interrupt/resume уже другой, чем в старых примерах. Все eino-типы держим за интерфейсами adev, чтобы обновление фреймворка не задевало формы и домен.

## 3. Матрица покрытия

| # | Потребность adev | eino | Статус | Что строим сами |
| --- | --- | --- | --- | --- |
| 1 | Пайплайн стадий 00–09 | Graph/Workflow | частично | Оркестратор, состояние пайплайна, переходы |
| 2 | Форма как строгий контракт | — | нет | JSON Schema, типы, валидатор, хэши, версии |
| 3 | Delta-формы (JSON Patch) | — | нет | Патч, материализация, проверка базы |
| 4 | Реестр объектов targetSystem | — | нет | `objectId`, жизненный цикл, транзакции |
| 5 | Проекции по ролям | — | нет | Allowlist JSON Pointer, context builder |
| 6 | Роль-агенты с tools | ADK ChatModelAgent | да | Промпты, tool ACL, `submit_form` |
| 7 | Агент не угадывает | — | нет | Tool-схема формы + validation-repair loop |
| 8 | TDD-петля 07–09 | Graph + tools | частично | Codegen catalog, sandbox, test runner, verdict parser |
| 9 | Задачи с одним владельцем | — | нет | Task engine, lease, дерево задач |
| 10 | Неопределённость → оператор | Interrupt/resume | частично | `QUESTION` store, API, resume, SLA |
| 11 | Оператор может прервать | `WithGraphInterrupt`, cancel | частично | Abort в оркестраторе, идемпотентность |
| 12 | Локальные + облачные модели | eino-ext models | да | Model gateway, профили, квоты, бюджет |
| 13 | 1000 req/day + $10/month | — | нет | Quota ledger, reservation, cost accounting |
| 14 | Минимальный контекст | — | нет | Проекции, лимиты токенов, запрет истории |
| 15 | Горизонтальное масштабирование | stateless graphs | частично | Очередь, lease, общий checkpoint store |
| 16 | ASD-STE100 / ACE | — | нет | Чекер, глоссарий, гейт на `Text` |
| 17 | Повторное использование | — | нет | Каталог codegen и технологий, форма-гейт |
| 18 | Аудит и стоимость | Callbacks | частично | Event log, token/cost ledger |
| 19 | Operator UI | A2UI (опционально) | нет | HTTP API + UI |
| 20 | Jira / Git / CI | commandline/MCP | частично | Адаптеры, allowlist |

## 4. Архитектурные решения

### D1. Одна стадия = один eino-граф

Не строим один гигантский граф на весь пайплайн. Причины:

- change-маршрут и дерево задач динамические;
- чекпоинт гигантского графа растёт и ломается при обновлении eino;
- стадии масштабируются независимо;
- падение задачи не должно перезапускать анализ.

Оркестратор в БД запускает граф стадии. Граф возвращает `formId` и `status`. Переход делает оркестратор.

### D2. Формы — состояние, граф — только ссылки

В state графа: `targetSystemId`, `pipelineId`, `inputRefs`, `formId`, `attempt`. Тела форм лежат в Postgres. Это даёт маленькие чекпоинты и переживает рестарт.

### D3. Долгое ожидание оператора не держит чекпоинт

`QUESTION` завершает run стадии, сохраняет draft и `operator_request`. Ответ оператора создаёт новую версию формы и новый run. Eino interrupt используем для коротких approval внутри run, не для ожидания в днях.

### D4. Агент сдаёт форму инструментом

Каждая форма публикуется как tool `submit_<formType>` с JSON Schema. Агент обязан вызвать tool. Валидатор возвращает ошибки как результат tool, агент исправляет. Максимум 2–3 попытки, затем `QUESTION`. Свободный текст ответом не принимается.

### D5. Контекст собирает adev, не агент

Context builder:

1. Берёт `inputs` формы.
2. Применяет проекцию роли.
3. Добавляет задачу, инструкцию стадии, список tools.
4. Считает токены под конкретную модель.
5. Не добавляет историю чата и полные формы.

Агент не имеет доступа к БД. Только tools.

### D6. Model gateway — обёртка над `model.BaseChatModel`

Профили: `classifier`, `worker`, `heavy`, `cloud_free`, `cloud_paid`, `backup_slow`.

Gateway делает:

- выбор профиля по стадии и типу задачи;
- проверку квоты и бюджета **до** вызова;
- резервирование и списание токенов;
- failover только на модель с доступной квотой;
- семафор на локальную модель;
- проверку контекстного окна;
- запись `model_calls` (tokens, cost, latency, model).

Eino failover используем для транзиентных ошибок внутри выбранного профиля. Квоты — наши.

### D7. Tools — именованные, с ACL, без shell для агента

Агенту доступны только tools из реестра для его роли и стадии. `commandline` и `mcp` не даём напрямую. Внутренний executor запускает команды в sandbox (Docker, без сети, лимиты CPU/RAM/времени).

### D8. Postgres-only MVP

Postgres хранит формы, реестр, runs, checkpoints, operator requests, события, стоимость. Redis не обязателен. Очередь — таблица `runs` с `FOR UPDATE SKIP LOCKED`. Если понадобятся cron/сложные retry, добавим River; Temporal не берём на MVP.

### D9. Языковой гейт — валидатор, не промпт

Каждое поле `Text` проходит чекер. Провал — ошибка tool, не «попроси модель переписать». Глоссарий проекта расширяет STE100: `pipeline`, `schema`, `container`, `module`, `artifact` и т.п. ACE — отдельный режим для нормативных `Statement`.

### D10. Реестр форм — источник, схемы — генерация

Markdown-реестр остаётся HLD-документом. Runtime-схемы (`json-schema/*.json`, Go-типы) генерируются из него и проверяются в CI на расхождение. Ручная правка схем запрещена.

## 5. Чего не хватает: инвентарь

### P0 — без этого нельзя запустить одну стадию

| Компонент | Минимум | Готово когда |
| --- | --- | --- |
| `internal/forms` | JSON Schema, Go-типы, validate, canonical hash, JSON Patch, refs | Форма BRD валидируется и сохраняется |
| `internal/registry` | `objectId`, kind, status, introducedBy, транзакционная выдача | LLD создаёт `planned`-объект, VERDICT делает `active` |
| `internal/projection` | allowlist JSON Pointer по роли | Dev не видит BRD/HLD, SysA не видит `module.code` |
| `internal/pipeline` | pipeline/run state, lease, retry, идемпотентность | Два воркера не берут один run |
| `internal/store` | Postgres-миграции, формы, runs, checkpoints, events | Рестарт процесса не теряет run |
| `internal/agents` | роль-агент, промпт стадии, `submit_form`, retry на ошибках валидации | Stage 00 даёт валидную BRD |
| `internal/context` | сборка контекста, лимит токенов | Контекст ≤ окна модели, без истории |
| `internal/modelgw` | один локальный профиль + один cloud | Стадия работает на Ollama/vLLM |
| `internal/httpapi` | create pipeline, get run, list forms, answer question, abort | Оператор управляет пайплайном |

### P1 — TDD-петля

| Компонент | Минимум | Готово когда |
| --- | --- | --- |
| `internal/tools/codegen` | каталог генераторов, выбор по `TargetRef` | ImplementationSpec не содержит кода |
| `internal/sandbox` | Docker, repo copy, no network, limits | Тест и codegen не трогают хост |
| `internal/tools/test` | `go test -json`, парсер результатов | VERDICT заполняется из отчёта |
| `internal/tools/git` | clone, branch, diff, commit | Изменения изолированы в ветке |
| `internal/tdd` | граф 07→08→09→08.R | Fail возвращается в 08.R, pass закрывает задачу |
| `internal/langcheck` | STE100 + глоссарий, режим ACE | `Text` с запрещённым словом не проходит |
| `internal/operator` | QUESTION store, resume, approve, abort | Ожидание и ответ переживают рестарт |

### P1 — модели и деньги

| Компонент | Минимум | Готово когда |
| --- | --- | --- |
| `internal/modelgw/quota` | token bucket per provider/model/day | 1001-й запрос отклоняется |
| `internal/modelgw/budget` | ledger, reservation, hard stop | Превышение $10/month блокирует paid |
| `internal/modelgw/router` | профили, выбор по стадии | Классификация идёт на малой модели |
| `internal/obs` | eino callbacks → OTel/Prometheus | Виден путь формы и стоимость run |
| `internal/audit` | append-only events | Любой переход и ответ оператора восстановимы |

### P2 — change-маршрут и эксплуатация

| Компонент | Минимум | Готово когда |
| --- | --- | --- |
| `internal/rules` | routing.yaml, tech.yaml, evaluator | Route A/B выбирается по правилу, не по модели |
| `internal/change` | CR, impact, delta-формы | Route A и B проходят на тестовом targetSystem |
| UI | pipeline list, form viewer, diff, question, approve, abort, cost | Оператор не пользуется curl |
| Адаптеры | Jira, GitHub/GitLab, CI | Task Plan уходит в Jira, PR создаётся |
| `internal/eval` | golden forms, fake model, E2E | Регрессии промптов и форм ловятся в CI |

## 6. Предлагаемая структура

```
ai-dev-system/
  cmd/adev/                 # api, worker, migrate
  internal/
    forms/                  # schema, types, validate, patch, hash
    registry/               # targetSystem objects
    projection/             # role -> pointers
    pipeline/               # state machine, runs, leases
    store/                  # postgres
    agents/                 # role agents, prompts
    context/                # context builder
    modelgw/                # profiles, quota, budget, router
    tools/                  # form, codegen, test, git, lang
    sandbox/                # docker exec
    tdd/                    # 07-09 graph
    rules/                  # routing, tech, codegen catalogs
    operator/               # question, approval, abort
    obs/                    # callbacks, metrics, tracing
    httpapi/
  migrations/
  schemas/                  # generated JSON Schema
  rules/
  docs/
  form-registry/
```

## 7. Минимальная модель данных

```
target_systems(id, name, repo_url, created_at)
pipelines(id, target_system_id, route, status, current_stage)
forms(form_id, form_version, form_type, pipeline_id, stage,
      status, payload jsonb, payload_hash, previous_form_version,
      produced_by, created_at)
form_refs(form_id, form_version, ref_form_id, ref_version, pointer)
object_registry(object_id, target_system_id, kind, name, path,
                container_id, status, fields jsonb, introduced_by)
runs(id, pipeline_id, stage, task_id, status, checkpoint_id,
     lease_until, attempts, last_error)
tasks(id, pipeline_id, parent_id, owner, status, form_id)
operator_requests(id, pipeline_id, form_ref, pointer, question,
                  options jsonb, blocking, status, answer jsonb)
model_calls(id, run_id, profile, model, tokens_in, tokens_out, cost, latency_ms)
events(id, pipeline_id, run_id, type, payload jsonb, created_at)
checkpoints(key, value bytea, updated_at)
```

Формы append-only. `payload_hash` — канонический JSON. `form_refs` позволяет проверять трассировку без парсинга prose.

## 8. Модель исполнения

```
worker:
  lease run (SKIP LOCKED)
  load input forms + projection
  compile stage graph (cached)
  Invoke with checkpoint_id
  validate output form
  persist form + event
  transition pipeline or create operator_request

operator answers QUESTION:
  create new form version
  mark request answered
  enqueue new run

abort:
  cancel ctx / WithGraphInterrupt
  mark run aborted
  leave forms intact
```

TDD-задача:

```
07 TEST_CONTRACT (QA agent)
08 IMPLEMENTATION_SPEC (Dev agent)
tools: codegen -> test -> report
09 VERDICT (QA agent)
fail -> 08.R REFACTOR_SPEC -> 09
pass -> task done, registry active
```

## 9. План по этапам

Оценка для одного опытного Go-разработчика. Этапы можно частично параллелить.

| Этап | Срок | Результат | Exit criteria |
| --- | --- | --- | --- |
| M0. Каркас | 1 нед | `go.mod`, конфиг, Postgres, миграции, логи, OTel, fake model, eino hello graph | `go test ./...` зелёный, граф запускается |
| M1. Формы | 2 нед | схемы, типы, валидатор, hash, patch, проекции, реестр объектов | BRD/LLD/VERDICT валидируются, проекции тестами |
| M2. Greenfield | 3 нед | стадии 00–06, агенты, `submit_form`, context builder, orchestrator, QUESTION, checkpoint store | Один targetSystem доходит до Task Plan |
| M3. TDD | 3 нед | 07–09, codegen catalog, sandbox, test runner, verdict, refactor loop | Тест падает → код/конфиг → тест проходит |
| M4. Модели | 2 нед | gateway, профили, квоты, бюджет, failover, cost | Локальная модель + cloud + backup, лимиты не пробиваются |
| M5. Change | 2 нед | CR, impact, routing rules, delta SRS/HLD/LLD | Route A и B на тестовом кейсе |
| M6. Эксплуатация | 3 нед | UI, аудит, Jira/Git/CI, eval, нагрузка | Оператор работает без curl, есть дашборд |

Критический путь: **M1 → M2 → M3**. M4 можно начинать после M0. M5 зависит от M2/M3.

### Первый вертикальный срез

Не строить всё сразу. Первый срез — **только stage 00**:

1. `USER_INPUT` → `BRD`.
2. Один локальный модель-профиль.
3. `submit_brd` tool.
4. Валидация схемы + языка.
5. Сохранение в Postgres.
6. Просмотр формы и запуск из CLI/HTTP.

Это доказывает связку eino + форма + БД + модель. После этого каждая стадия — копия паттерна.

## 10. Риски

| Риск | Митигация |
| --- | --- |
| eino API меняется | Пин версии; eino-типы только в `internal/agents`, `internal/tdd`, `internal/modelgw` |
| Чекпоинты несовместимы между версиями | Короткие run, state = ссылки; не ждать оператора внутри чекпоинта |
| Локальная модель плохо вызывает tools | Профиль `worker` только для tool-capable моделей; fallback JSON + validator + repair |
| 32k контекст | Проекции, лимит, отказ с `QUESTION`; без молчаливого обрезания |
| Codegen не покрывает стек | Каталог генераторов; `manualException` только с approval оператора |
| Sandbox escape | Docker без сети, non-root, read-only root, лимиты, allowlist путей |
| Перерасход бюджета | Reservation до вызова, hard stop, отдельный ledger |
| Оператор — бутылочное горлышко | Batch вопросов, defaults, SLA, эскалация |
| Формы и код расходятся | Генерация схем из реестра + CI diff |
| STE100 словарь не покрывает домен | Глоссарий проекта + тесты; ACE — отдельный режим |
| Агент пытается писать код | Нет shell tool; только codegen tools; `manualException` с approval |

## 11. Открытые решения

1. **vLLM или Ollama** для production local. Предложение: Ollama для разработки, vLLM (OpenAI-compatible) для нагрузки.
2. **Своя очередь или River.** Предложение: своя таблица `runs` на MVP.
3. **UI:** templ + HTMX или React. Предложение: HTMX, меньше кода.
4. **ACE-парсер:** внешний Attempto или шаблоны + лексический чекер. Предложение: начать с шаблонов, парсер позже.
5. **Jira:** обязательна ли на M2 или M6.
6. **Multi-tenant:** один targetSystem на инстанс или несколько. Влияет на изоляцию и ключи.
7. **Формат схем:** JSON Schema 2020-12. Подтвердить.

## 12. Что не делать

- Не использовать `DeepAgent` для маршрутизации стадий.
- Не давать агенту `commandline`/shell.
- Не хранить формы только в чекпоинтах.
- Не строить один граф на весь пайплайн.
- Не позволять агенту выбирать следующую стадию.
- Не отдавать агенту полную форму вместо проекции.
- Не надеяться на eino в квотах, стоимости и языке.
