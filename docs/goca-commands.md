# GOCA: команды для подготовки adev

Источник: [goca commands](https://sazardev.github.io/goca/commands/), версия v1.18.x.
Цель: получить рабочий каркас Clean Architecture для adev, на который накладываются eino-графы, формы и песочница.

Правило безопасности: любой генерирующий вызов сначала с `--dry-run`. Флаги `--dry-run`, `--force`, `--backup` есть у всех 12 генераторов.

## 0. Предпосылки

```bash
go version          # нужен Go 1.21+ (сам goca собирается на 1.25.x)
git --version
```

## 1. Установка goca

```bash
go install github.com/sazardev/goca@latest
goca version
```

Windows: `%USERPROFILE%\go\bin` должен быть в `PATH`.

## 2. Создание проекта adev

```bash
cd D:\stimy\Documents\dev\AIAutomat\ai-dev-system

goca init --list-templates
goca init adev --module github.com/<owner>/adev --database postgres --template enterprise --dry-run
goca init adev --module github.com/<owner>/adev --database postgres --template enterprise

cd adev
go mod tidy
go build ./...
```

`goca init` создаёт `ai-dev-system/adev/` и сам делает `git init` + первый коммит. Документация и `form-registry/` остаются вне этого модуля.

Шаблон: `enterprise` (auth, метрики, Docker, K8s, CI, покрытие 85%). Альтернатива — `microservice`, если нужны доменные события и аудит.

Заметки:
- `--api grpc|graphql` только пишется в конфиг, scaffolding не генерируется. Только REST.
- ID по умолчанию `uint`, payload JSONB и ULID добавляются вручную.

## 3. Конфигурация

```bash
goca config show
goca config validate
```

Правки в `.goca.yaml`: `module`, `database.type: postgres`, `handlers: [http, cli, worker]`.

## 4. Свои шаблоны (до генерации фич)

```bash
goca template init
goca template list
```

В `.goca/templates/` привести шаблоны к стилю adev: snake_case файлы, godoc, `context.Context`, без GORM-специфики, ULID вместо `uint`.

## 5. Middleware

```bash
goca middleware adev --types cors,logging,auth,rate-limit,recovery,request-id,timeout --dry-run
goca middleware adev --types cors,logging,auth,rate-limit,recovery,request-id,timeout
```

Внешние зависимости добавляются вручную: `github.com/golang-jwt/jwt/v5`, `golang.org/x/time`, `github.com/google/uuid`.

## 6. Домен adev (control plane)

Поля взяты из модели данных в `docs/implementation-plan.md`.
Сначала dry-run, потом реальный вызов.

```bash
# Ядро пайплайна
goca feature TargetSystem --fields "name:string,repoURL:string,description:string" --validation --dry-run
goca feature TargetSystem --fields "name:string,repoURL:string,description:string" --validation

goca feature Pipeline --fields "targetSystemID:int,route:string,status:string,currentStage:string" --validation --handlers "http,cli,worker"

goca feature FormRecord --fields "formID:string,formType:string,formVersion:int,stage:string,status:string,payloadHash:string,producedBy:string,pipelineID:int" --validation

goca feature Run --fields "pipelineID:int,stage:string,status:string,checkpointID:string,attempts:int,lastError:string" --handlers "http,worker"

goca feature OperatorRequest --fields "pipelineID:int,pointer:string,question:string,blocking:bool,status:string" --validation --handlers "http,cli"
```

```bash
# Остальная модель
goca feature FormRef --fields "formID:string,formVersion:int,refFormID:string,refVersion:int,pointer:string"

goca feature ObjectEntry --fields "targetSystemID:int,kind:string,name:string,path:string,containerID:string,status:string" --validation

goca feature Task --fields "pipelineID:int,parentID:int,owner:string,taskType:string,status:string,contractRef:string" --handlers "http,worker"

goca feature ModelCall --fields "runID:int,profile:string,model:string,tokensIn:int,tokensOut:int,cost:float64,latencyMs:int"

goca feature AuditEvent --fields "pipelineID:int,runID:int,eventType:string,createdAt:time.Time"
```

`FormRecord.payload` (JSONB) и `ObjectEntry.fields` (JSONB) добавляются в сгенерированные структуры вручную.

## 7. Только нужные слои, если feature избыточен

```bash
goca entity ObjectEntry --fields "kind:string,name:string,path:string,status:string" --validation --business-rules --timestamps
goca usecase PipelineService --entity Pipeline --operations "create,read,update,list" --dto-validation
goca repository Run --database postgres --transactions
goca handler Run --type worker
```

## 8. Контракты, моки, интеграционные тесты

```bash
goca interfaces FormRecord --all
goca interfaces Pipeline --all
goca interfaces Run --all

goca mocks FormRecord
goca mocks Pipeline
goca mocks Run

goca test-integration FormRecord --container --dry-run
goca test-integration FormRecord --container
goca test-integration Pipeline --container
```

`--container` поднимает Postgres через Testcontainers. Без него используется in-memory SQLite.

## 9. Сообщения и константы

```bash
goca messages FormRecord --errors
goca messages FormRecord --responses
goca messages Pipeline --constants
goca messages Run --constants
```

## 10. DI и маршруты

```bash
goca integrate --all --dry-run
goca integrate --all
```

Или точечно:

```bash
goca di --features "TargetSystem,Pipeline,FormRecord,Run,OperatorRequest" --database postgres
```

## 11. CI

```bash
goca ci --with-docker --dry-run
goca ci --with-docker
```

Go-версия берётся из `go.mod`, Postgres-сервис — из `.goca.yaml`.

Позже, при появлении тега `v*`:

```bash
goca ci --with-deploy
```

## 12. MCP для кодового ассистента

```bash
goca mcp-server --print-config vscode    # GitHub Copilot
goca mcp-server --print-config cursor
goca mcp-server --print-config claude
goca mcp-server --print-config zed
```

Даёт ассистенту 16 tools (`goca_feature`, `goca_entity`, `goca_analyze`, ...) и ресурсы `goca://config`, `goca://structure`.

## 13. Проверка здоровья

```bash
goca doctor
goca doctor --fix

goca analyze
goca analyze --security --fail-on-warn
goca analyze --arch --output json
```

`analyze` проверяет границы слоёв, OWASP A03 (SQL-инъекции, секреты, `unsafe`), стандарты Go, тесты и зависимости. Exit code 1 — есть провалы, 2 — предупреждения с `--fail-on-warn`.

## 14. Обновление

```bash
goca upgrade
go install github.com/sazardev/goca@latest
```

## 15. После goca: то, что он не генерирует

```bash
mkdir -p internal/forms internal/projection internal/pipeline internal/modelgw
mkdir -p internal/agents internal/context internal/tools internal/sandbox
mkdir -p internal/tdd internal/rules internal/operator internal/obs
mkdir -p schemas rules migrations
```

Зависимости eino:

```bash
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/ollama@latest
go get github.com/cloudwego/eino-ext/components/model/openai@latest
go get github.com/cloudwego/eino-ext/components/model/openrouter@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest
```

Вручную пишутся:

- form engine: JSON Schema, Go-типы, validate, canonical hash, JSON Patch, refs;
- реестр объектов: `objectId`, lifecycle `planned → active → removed`;
- проекции ролей: allowlist JSON Pointer;
- оркестратор: runs, lease, retry, переход стадий;
- `CheckpointStore` на Postgres для eino;
- context builder, `submit_form` tool, validation-repair loop;
- model gateway: профили, квоты 1000/day, бюджет $10/month, failover;
- codegen catalog, Docker-sandbox, `go test -json` parser, verdict;
- языковой чекер STE100/ACE и глоссарий;
- rulebook маршрутизации A/B и правила активации объектов.

## 16. Соответствие слоёв

| goca | adev |
| --- | --- |
| `internal/domain` | Go-типы форм, пайплайна, реестра, задач |
| `internal/usecase` | сервисы: pipeline, form, registry, quota |
| `internal/repository` | Postgres-хранилище |
| `internal/handler/http` | operator API |
| `internal/handler/cli` | локальные операции |
| `internal/handler/worker` | воркеры стадий и задач |
| `internal/di` | сборка зависимостей |

Пакеты `internal/agents`, `internal/tdd`, `internal/modelgw`, `internal/tools` живут рядом; `goca analyze` их не ломает, он проверяет только четыре CA-слоя.

## 17. Два предупреждения

1. **`goca analyze` строгий.** Правила `no-todos`, `exported-docs`, `no-empty-files`, `handler-no-repo` могут валить CI на промежуточном коде. Запускать с `--baseline`-подобным подходом нельзя, флага нет — либо чинить, либо не включать в CI до стабилизации.
2. **GORM vs pgx.** `--database postgres` тянет GORM и `database/sql`-шаблоны. Для форм с JSONB и транзакционных lease это придётся заменить на pgx. Goca даёт каркас, не финальный store.
