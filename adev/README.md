# adev

Агентная система на Go, которая строит софт по маршруту форм: агенты заполняют формы
(`USER_INPUT` → `BRD` → `SRS` → `HLD` → `LLD` → …), гейты проверяют каждую, а спорные
места уходят оператору вопросами. 
пользовательский интерфейс оператора — React-приложение
в `../frontend`.

Документы системы лежат рядом с модулем: `../docs/SysReq.md`, `../docs/Arhitecture.md`,
реестр форм — `../form-registry/`.

## Слои

```
internal/
├── domain/      сущности, правила, гейты — без I/O и context.Context
├── usecase/     сервисы, DTO, порты (интерфейсы)
├── repository/  pgx-адаптеры, реализуют порты usecase
├── handler/     http- и cli-адаптеры
├── di/          корень композиции
└── middleware/  http-middleware
```

Порт объявляет `usecase`; `repository` его реализует, а не наоборот. Хранилище —
PostgreSQL через pgx. Миграции — golang-migrate: `migrations/NNN_name.up.sql` и
`NNN_name.down.sql`, нумерация только вперёд. Схему не менять через код.

## Быстрый старт

Нужны Go, Docker и make (на Windows — WinGet Links).

```bash
make deps          # скачать модули
make dev-db        # поднять БД и дождаться healthcheck
make migrate-up    # применить миграции
make run           # сервер на :8080 с настройками из .env
```

Скопируйте `.env.example` в `.env` и заполните:

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` — база;
- `OUPENROUTER_APIKEY` — ключ модели. Имя с опечаткой это исторически
  (см. `pkg/env/env.go`), менять его означает ломать все существующие `.env`. 

Health check: `curl http://localhost:8080/health`.

## Интерфейс оператора

React + TypeScript + Vite в `../frontend`. Три вкладки: Target System (обзор),
Pipeline (схема стадий, карточки прогонов, вопросы оператору), Настройки
(targetSystem, workspace, агенты, проверка модели). Разработка и сборка:

```bash
make ui-install    # npm install, один раз после клона
make ui-dev        # Vite на :5173, проксирует /api на :8080 — CORS не нужен
make ui-build      # собрать ../frontend/dist; make run отдаёт его на /app/
```

`frontend/src/api.ts` — единственный HTTP-слой, каждый запрос попадает в журнал.
Внизу экрана — сворачиваемая панель: «Журнал запросов» (что сделал браузер) и
«Логи сервера» (`GET /api/v1/logs`).

## Пайплайн

- Стадии и владельцы — `internal/pipeline/stages.go`; маршрут принадлежит реестру,
  агент не выбирает следующую стадию.
- Стадия не стартует без готовности: workspace и валидный конфиг агента на каждую роль.
- Запуск (`POST /stages/{stage}/run`) отвечает 202 сразу, цепочка агентов работает
  в фоне; состояние — в таблице `runs`. Закрытая вкладка прогон не отменяет,
  при старте сервер возобновляет зависшие `running`.
- Гейт бракует форму → форма падает в `waiting_operator` и выпускает вопрос
  оператору (`GET /api/v1/operatorrequests`). Ответ (`POST /api/v1/operatorrequests/answer`)
  переписывает payload черновика на месте, без новой версии. После `approved`
  форма неизменяема: исправление — новая `formVersion`.
- `apiKey` не покидает сервер: в ответах отдаётся только `hasApiKey`.

## Проверка изменений

```bash
make fmt && make vet && make test
```

- `make test-tdd` — acceptance-тесты из `docs/refactor-plan.md`.
- `make test-db` — тесты pgx-хранилища. Они делают `TRUNCATE forms`, поэтому
  требуют отдельную тестовую базу в `ADEV_TEST_DATABASE_URL`; на dev-базу из
  `.env` их направлять нельзя.
- `make lint` и `make sec-scan` — golangci-lint и gosec (`make tools` ставит их вместе с migrate и swag).
- `make ci-act ACT_JOB=unit` запускает GitHub Actions job локально через act;
  образ нужен свой (`make act-image`): у стандартного нет ни git, ни сертификатов, ни gcc.

## Логи

`logger.Init` пишет каждую строку в stdout, в кольцо в памяти (500 строк, оттуда
их читает консоль) и в файл `logs/adev.log` (меняется через `LOG_FILE`). Файл
нужен, потому что фоновый процесс без консоли иначе не оставляет следов.
`make logs` следит за файлом.

## Команды

`make help` показывает полный список. Частые:

| Команда | Что делает |
|---|---|
| `make run` | сервер с настройками из `.env` |
| `make test` / `make test-race` | тесты; `-race` требует cgo |
| `make migrate-up` / `make migrate-down` | применить / откатить миграцию |
| `make api-docs` | перегенерировать Swagger-доки (swag) |
| `make build-prod` | статический linux-бинарник |
| `make compose-up` | приложение и БД вместе |

## Частые проблемы

**«connection refused»** — не запущен Postgres: `make dev-db`.

**«database not configured»** — не заполнены `DB_*` в `.env`.

**health check показывает «degraded»** — база не отвечает; проверьте `.env` и порт 5432.

**«command not found: migrate»** — `make migrate-install`.

**модели с суффиксом `-free` через API не работают** — OpenCode Console отвечает
403 `FreeTierError`; бесплатный тариф живёт только в клиенте OpenCode.
