# adev — правила работы в этом репозитории

## Что это

adev — агентная система на Go + eino, которая строит софт по маршруту форм.
Документы: `../docs/SysReq.md`, `../docs/Arhitecture.md`, `../docs/implementation-plan.md`,
реестр форм: `../form-registry/`.

## Слои (Clean Architecture, каркас goca)

```
domain/      сущности, value objects, правила          — без I/O и context.Context
usecase/     сервисы, DTO, порты (интерфейсы)          — без SQL и HTTP
repository/  pgx-адаптеры, реализуют порты usecase
handler/     http / cli / worker адаптеры
di/          корень композиции
middleware/  http-middleware
```

Правила: новый код только в этих слоях. Самодельные пакеты вне слоёв не заводить.
Порт объявляет `usecase`; `repository` его реализует, а не наоборот.
Пакет `internal/constants` удалён — константы живут рядом с тем, кто их использует.

## Стек и запреты

- Хранилище — **PostgreSQL через pgx**. GORM, `lib/pq`, `go-sql-driver/mysql`, `sqlite` удалены.
  Не возвращать их. Запросы только параметризованные (`$1`, `$2`).
- Миграции — golang-migrate, файлы `migrations/NNN_name.up.sql` / `.down.sql`, нумерация только вперёд.
  Применять: `make migrate-up`. Схему не менять через код.
- Домен, репозитории и usecase сгенерированы `goca`; правки в них делать в стиле окружающего кода.

## Makefile — точка входа

```
make help          список целей
make build / test / vet / fmt
make dev-db        поднять БД и дождаться healthcheck
make migrate-up    применить миграции
make run           запустить сервер с настройками из .env
make tools         поставить migrate, golangci-lint, gosec, swag
make ui-install    npm install в ../frontend, один раз после клона
make ui-dev        Vite на 5173, проксирует /api на сервер
make ui-build      собрать ../frontend/dist, make run отдаёт его на /app/
```

`make` доступен через WinGet Links. `make test-race` требует cgo.

Локальный запуск GitHub Actions через act:

```
make act-image            один раз собрать образ раннера (~700 МБ)
make ci-act-dry ACT_JOB=unit    план без контейнера
make ci-act ACT_JOB=unit        тот же job локально, ~80 с
```

Образ обязателен: образ act по умолчанию без git, без корневых сертификатов и без gcc.
`ci-act` сам переходит в корень репозитория, потому что workflows и `.actrc` лежат там, а не в `adev/`.
Postgres-job может попасть в ваш Postgres на 5432: запускайте его только когда там тестовая БД,
либо используйте `make ci` и `make test-db`.

## Frontend

UI оператора — React + TypeScript + Vite в `frontend/`, рядом с модулем Go. Пока один экран:
targetSystem → окружение → готовность → пайплайн → USER_INPUT → запуск стадии → формы.

- Типы в `frontend/src/api.ts` повторяют DTO Go; единственный HTTP-слой, каждый запрос
  попадает в журнал на экране — по нему и отлаживаем.
- Разработка: `make ui-dev` → http://localhost:5173, Vite проксирует `/api` на :8080,
  поэтому CORS не нужен. Сборка: `make ui-build` → `frontend/dist`, `make run` отдаёт её на `/app/`.
- Библиотеки без нужды не добавлять: роутер, UI-кит и кеш запросов — когда экранов станет больше.

## Формы

- Источник правды — markdown в `../form-registry/`. Runtime-типы — `internal/forms`.
- Конверт обязателен: `formId`, `formType`, `formVersion`, `status`, `mode`, `inputs`, `payloadHash`.
- Форма неизменяема после `approved`. Исправление — новая `formVersion` с `previousFormVersion`.
- Delta — `mode: "delta"` + JSON Patch (RFC 6902), не новый тип формы.
- Агент сдаёт форму инструментом, а не текстом. Валидатор возвращает ошибки, агент исправляет.
- Свободный текст только в `Text`/`Statement`/`Scenario`; язык ASD-STE100/ACE.
- `objectId` из реестра объектов, а не путь, для всего, что уже существует.

## Пайплайн

- Стадии и владельцы — `internal/pipeline/stages.go`. Маршрут принадлежит реестру, **агент не выбирает следующую стадию**.
- Стадия не стартует без готовности: workspace + валидный конфиг агента на каждую роль (`internal/settings`).
- `apiKey` не покидает сервер: `json:"-"`, в ответах только `hasApiKey`.

## Проверка изменений

```
make fmt && make vet && make test
```

Postgres-тесты включаются только переменной `ADEV_TEST_DATABASE_URL` на отдельную тестовую БД:
они делают `TRUNCATE forms`, и на dev-БД из `.env` их запускать нельзя.

## Правила ответа

- Минимальная правка, которая полностью решает задачу. Без абстракций «на будущее».
- Не добавлять зависимость без обоснования.
- Комментарий — только там, где код не объясняет «почему».
- Известное ограничение помечать `shortcut: <ограничение>, <когда исправлять>`.
