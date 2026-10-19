# Реестр динамических объектов targetSystem

Один `targetSystemId` = один реестр. Объекты появляются, меняются и удаляются в ходе работы adev, поэтому реестр — не статическая схема, а журнал.

## Зачем

Форма должна спрашивать про конкретный объект, а не про «модуль вообще». Без реестра агент отвечает текстом, и предметность теряется. Реестр даёт каждому объекту `objectId`, на который ссылаются все формы.

## Запись реестра

| field | type | req | rule |
| --- | --- | --- | --- |
| `objectId` | string | R | `OBJ-<kind>-<n>`. Выдаётся один раз, не меняется. |
| `targetSystemId` | string | R | Реестр-владелец. |
| `kind` | enum | R | `container`, `layer`, `package`, `module`, `table`, `broker`, `test`, `config`, `pipeline`, `env`, `doc`. |
| `name` | string | R | Имя объекта в targetSystem. |
| `path` | string | O | Физический путь. Может измениться без смены `objectId`. |
| `containerId` | string | O | Владелец для `layer`, `package`, `module`. |
| `status` | enum | R | `planned`, `active`, `removed`. |
| `fields` | object | O | Поля из `targetObjects.md`: `boundedContext`, `aggregates`, `contracts`, `tests`, `envConfig`. |
| `introducedBy` | `Ref` | R | Форма, которая создала запись. |
| `changedBy` | `Ref[]` | R | Формы, которые меняли запись. |
| `relations` | string[] | O | `objectId` связанных объектов. |

## Правила

1. `objectId` не переиспользуется после `status: removed`.
2. `planned` ставит `HLD` или `LLD`. `active` ставит `VERDICT` с `pass`. `removed` ставит `ADR`.
3. Форма ссылается на `objectId`. Путь в форме — только для нового объекта, которого ещё нет в реестре.
4. Новый объект без записи в реестре = `Gap`, а не догадка агента.
5. `test` создаёт `TEST_CONTRACT`. `config`, `pipeline`, `env`, `table` создаёт `IMPLEMENTATION_SPEC`.
6. Один объект имеет одного владельца-контейнера. Кросс-владение запрещено.

## Шаблон записи

```jsonc
{
  "objectId": "OBJ-module-014",
  "targetSystemId": "TGT-01J9Z8K000",
  "kind": "module",
  "name": "token_validator",
  "path": "internal/auth/token_validator.go",
  "containerId": "OBJ-container-002",
  "status": "planned",
  "fields": { "contracts": ["CT-44"] },
  "introducedBy": { "formId": "FRM-...", "formType": "LLD", "formVersion": 1, "pointer": "/payload/contracts/3" },
  "changedBy": [],
  "relations": ["OBJ-test-021"]
}
```

## Кто читает

Все формы. Проекция роли решает, какие `fields` попадают в контекст агента.
