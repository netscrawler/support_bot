# Design

## Context

Как устроен сбор сейчас: `collector.Collector` по `models.Card.Type` (`mb`/`appmetrica`/`jira`) выбирает `DataFetcher.Fetch(ctx, target, params)`, а `target` берётся из `card_uuid`. Запросы хранятся в таблице `queries` (`card_uuid`, `title`, `q_type`, `params jsonb`), карточки читает `ListCardsByReportID`. Те же коллекторы отдаются в Lua через `luastd.NewCollector(map[string]DirectCollector)` (`internal/app/module_report.go`). Пул Postgres уже есть (`internal/postgres`), `pgx/v5` и `pgxmock` подключены.

Мотивация описана в proposal.md, требования — в `specs/postgres-collector/spec.md`.

## Goals / Non-Goals

**Goals:**
- Pg-коллектор, который физически не может писать: защита в коде плюс второй слой на уровне соединения.
- Встроиться в существующий switch и Lua-плагин, не меняя `DataFetcher` и остальные коллекторы.

**Non-Goals:**
- Другие СУБД. Пока только Postgres, абстракцию «под любую БД» не делаем.
- Лимит строк в результате: у Metabase и Jira его тоже нет.
- Управление базами из бота или БД. Базы задаются только в конфиге.

## Decisions

### D1. Хранение: `card_uuid` = имя базы, SQL в новой колонке `queries.query`
Договорённость с пользователем. `params` содержит только подставляемые и изменяемые значения. Миграция `006_add_query_text.sql`: `alter table queries add column query text;` (nullable, у не-pg запросов остаётся NULL). Номер 005 занят файлом, на который уже ссылается `sqlc.yaml`.
- Альтернатива 1: `q_type = 'pg'`, `card_uuid` = SQL, имя базы в `params`. Отклонена пользователем: `params` должны быть только подставляемыми.
- Альтернатива 2: `q_type = 'pg:analytics'`. Отклонена, потому что ломает простой switch по типу.

### D2. Дедупликация: ключ `(card_uuid, title, query)`
`FindQueryIDByUUIDAndTitle` получает условие `and query is not distinct from $3`. Без него два pg-запроса к одной базе с одинаковым title, но разным SQL, склеятся, и второй SQL тихо потеряется. `is not distinct from` сохраняет прежнее поведение для не-pg запросов, у которых `query` равен NULL.

### D3. Отдельный интерфейс вместо расширения `DataFetcher`
В пакете `collector` появляется `SQLFetcher interface { Fetch(ctx, db, sql string, params map[string]string) ([]map[string]any, error) }`. В switch добавляется ветка `case models.CollectTypePostgres: data, err = c.pg.Fetch(ctx, crd.CardUUID, crd.Query, crd.Params)`. Конструктор `NewCollector` принимает ещё один аргумент. Mockery генерирует мок автоматически, потому что для `internal/collector` в `.mockery.yaml` стоит `all: true`.
- Альтернатива: передавать во все `Fetch` весь `models.Card`. Отклонена: это задело бы все коллекторы, Lua и моки ради одного нового.

### D4. Только чтение: два слоя
1. **В коде, всегда, без возможности отключить.** `BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})`, затем `SET LOCAL statement_timeout = <ms>`, затем `Query(ctx, sql, pgx.NamedArgs(params))`, и в конце `defer tx.Rollback(ctx)`. `Commit` не вызывается никогда. Postgres сам отклоняет запись, DDL и `nextval()`. pgx работает по extended protocol, поэтому за один вызов уходит ровно один стейтмент, и `select 1; delete ...` даёт ошибку.
2. **В конфиге: `read_only` (по умолчанию `true`).** В `RuntimeParams` пула ставится `default_transaction_read_only=on`. Это защита на случай будущей правки кода, где транзакция перестанет быть read-only. Значение `false` только снимает этот второй слой, первый слой продолжает работать.
- Регулярки по SQL отклонены: их легко обойти, а сервер проверяет надёжно.
- Остаётся риск функций с побочными эффектами через собственное соединение (`dblink`). Его закрывают права пользователя в DSN (см. Risks).

### D5. Конфиг
```yaml
databases:
  analytics:
    dsn: postgres://ro_user:pass@host:5432/db
    read_only: true          # default true
    statement_timeout: 30s   # default 30s
    max_conns: 4             # default 4
```
Тип `map[string]postgres.Config` — свой маленький `Config` в новом пакете `internal/collector/postgres`, не `internal/postgres.Config` (у того другой набор полей). Дефолты для полей map задаются в коде при создании пулов, потому что `env-default` cleanenv на значения map не действует. В `Config.LogValue` все DSN заменяются на `postgres://***`. Для этого map копируется, чтобы не испортить исходный конфиг.

### D6. Пулы и старт
`postgres.New(ctx, cfgs, log)` создаёт по `pgxpool` на каждую базу. `pgxpool.NewWithConfig` подключается лениво, `Ping` при старте только пишет warn и не валит бота. Неизвестное имя базы в `Fetch` даёт ошибку `unknown database "<name>"`. Пулы закрываются через `fx.Lifecycle` OnStop.

### D7. Форма результата: JSON-нормализация
`pgx.CollectRows(rows, pgx.RowToMap)` возвращает `pgtype.Numeric`, `time.Time`, `[16]byte` и подобные типы, а дальше (Lua `GoValueToLua`, шаблоны, DuckDB) ожидают JSON-подобные значения, как у Metabase. Поэтому результат один раз прогоняется через `json.Marshal`/`json.Unmarshal` в `[]map[string]any`. Числа становятся `float64`, даты — RFC3339-строками, то есть форма совпадает с Metabase.
- Альтернатива: свой конвертер типов. Отклонена: больше кода, а результат тот же.

### D8. Lua
У `luastd.CollectPlugin` появляется необязательное поле `pg` (интерфейс с той же сигнатурой, что `SQLFetcher`). Если оно задано, регистрируется `collector.pg.collect(db, sql, params)`, которая возвращает `rows, nil` или `nil, err`, как остальные. Таймаут контекста 5 минут, как у `luaCollect`, а `statement_timeout` базы обычно меньше. Через общий `directs` pg не идёт, потому что его сигнатура `(target, params)` не вмещает SQL.

### D9. Тесты
- Unit (`pgxmock`): проверяется последовательность BeginTx(ReadOnly) → `SET LOCAL statement_timeout` → Query с NamedArgs → Rollback без Commit, ошибка неизвестной базы и JSON-нормализация.
- Интеграционный тест на реальном Postgres гейтится переменной `PG_COLLECTOR_TEST_DSN` (без неё `t.Skip`). Он проверяет ключевые сценарии спеки: delete/create падают, `select 1; delete` падает, `pg_sleep` упирается в таймаут, `show default_transaction_read_only` = on. Testcontainers в проекте нет, поэтому новую зависимость не добавляем.
- `collector_test.go`: ветка `pg` вызывает `SQLFetcher` с `(CardUUID, Query, Params)`.
- `report_store_test.go` (`pgxmock`): `getOrCreateQuery` передаёт `query` в поиск и вставку.

## Risks / Trade-offs

- [Lua может читать всё, что видит пользователь DSN] → в DSN указывать отдельного read-only пользователя с правами только на нужные схемы. Это записывается в комментарий к `databases` в `config.example.yaml`. Перед archive обязателен `/security-review`.
- [`dblink` и другие функции с побочными эффектами обходят read-only транзакцию] → закрывается только правами пользователя в БД. Это записывается туда же.
- [`statement_timeout` можно переопределить внутри SQL через `set_config`] → в одном стейтменте это не помогает текущему запросу, а внешний ctx Lua (5 минут) и таймаут коллектора остаются. Принимаем.
- [JSON-нормализация теряет точность `numeric` больше float64 и превращает `bytea` в base64] → такая же семантика, как у Metabase. Принимаем.
- [`migrations/*` в `.gitignore`, из-за этого `005_create_public_reports.sql` не попал в ветку] → файл восстановлен из `a783c92`. `005` и `006` коммитятся через `git add -f`.

## Migration Plan

1. Накатить `006_add_query_text.sql` на существующую БД вручную (`docker-entrypoint-initdb.d` применяет его только на пустой БД).
2. Добавить секцию `databases` в рабочий конфиг. Без неё бот работает как раньше, pg-запросы падают с ошибкой «unknown database».
3. Откат: старый бинарник колонку `query` игнорирует. Саму колонку можно оставить.
