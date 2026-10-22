# Proposal

## Why

Сейчас данные для отчётов из баз Postgres забираются только через Metabase: под каждый запрос нужно заводить карточку, а скорость и доступность зависят от Metabase. Нужен прямой коллектор, который выполняет SQL напрямую в базе. Подключения к базам хранятся в конфиге бота, а в БД бота лежат только тип коннектора, имя базы и текст запроса.

## What Changes

- Новый тип запроса `pg`: в `card_uuid` хранится имя базы (ключ из конфига), SQL лежит в новой колонке `queries.query`, а `params` содержит только подставляемые значения (именованные параметры `@name`).
- Новая секция конфига `databases`: имя → `dsn`, `read_only`, `statement_timeout`, `max_conns`. DSN маскируется в логах.
- Запросы выполняются строго на чтение: каждый запрос идёт в read-only транзакции с `statement_timeout`. Если `read_only: true` (значение по умолчанию), соединение дополнительно открывается с `default_transaction_read_only=on`.
- Коллектор `pg` доступен из Lua: `collector.pg.collect(db, sql, params)`.
- Существующий запрос ищется по имени базы, заголовку и тексту SQL, так что разные SQL с одинаковым заголовком и базой не склеиваются.
- Миграция `006_add_query_text.sql` добавляет колонку `queries.query`. На существующей БД её надо накатить вручную.

## Capabilities

### New Capabilities
- `postgres-collector`: сбор данных отчёта прямым read-only SQL-запросом к Postgres-базе, описанной в конфиге, в том числе из Lua-скриптов.

### Modified Capabilities

## Impact

- Код: `internal/collector` (новый пакет `postgres`, ветка `pg` в `Collector`), `internal/models/report.go` (`Card.Query`, константа типа), `internal/config`, `internal/app/module_collectors.go`, `internal/app/module_report.go` (Lua), `internal/processor/lua/stdlib/collector.go`, `cmd/live-server/main.go`, `internal/store/report_store.go`.
- БД: `internal/db/queries/report.sql`, новая миграция и `sqlc.yaml`, затем перегенерация `sqlcgen`.
- Безопасность: Lua-скрипты получают возможность выполнять произвольный SQL на чтение в базах из конфига, поэтому перед archive обязателен `/security-review`.
- Зависимости: новых нет, используются уже подключённые `pgx/v5` и `pgxpool`.
