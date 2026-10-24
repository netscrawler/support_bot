# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test ./... && golangci-lint run`.
> Генерируемый код не править: `sqlcgen` — через `task generate`, моки — через `go tool mockery`.

## 1. Схема и хранение запроса

- [x] 1.0 Восстановлен `migrations/init.sql/005_create_public_reports.sql` из коммита `a783c92` (ветка `exportV2`), так что `task generate` снова проходит и `sqlcgen` не меняется. Файл попадает под `.gitignore` (`migrations/*`), поэтому при коммите его надо добавлять через `git add -f`.
- [x] 1.1 Создать `migrations/init.sql/006_add_query_text.sql` (`alter table queries add column query text;`) и добавить его в `schema` в `sqlc.yaml`. **На существующей БД накатить вручную.** Файл игнорируется git (`migrations/*`), при коммите нужен `git add -f`. Проверка: файл на месте, строка есть в `sqlc.yaml`.
- [x] 1.2 В `internal/db/queries/report.sql`: `ListCardsByReportID` дополнительно выбирает `q.query`, `CreateQuery` вставляет `query`, `FindQueryIDByUUIDAndTitle` получает условие `and query is not distinct from $3`. Затем `task generate`. Проверка: `task generate` без ошибок, `go build ./...`.
- [x] 1.3 `models.Card`: поле `Query string \`json:"query"\``, константа `CollectTypePostgres = "pg"`. В `internal/store/report_store.go` `mapCardRows` заполняет `Query` (NULL → ""), а `getOrCreateQuery` передаёт `query` в поиск и вставку (`""` → NULL). Тест в `report_store_test.go` (pgxmock): два pg-запроса с одним title и базой, но разным SQL создают два разных запроса. Проверка: `go test ./internal/store/...`.

## 2. Пакет pg-коллектора

- [x] 2.1 `internal/collector/postgres`: `Config{DSN, ReadOnly *bool, StatementTimeout, MaxConns}` с дефолтами в коде (read_only=true, 30s, 4), `New(ctx, map[string]Config, log)` с пулом на каждую базу (`RuntimeParams["default_transaction_read_only"]="on"` при read_only, Ping при старте только пишет warn), `Close()`. Тест: дефолты и `RuntimeParams` выставляются (через функцию сборки `pgxpool.Config`, без подключения). Проверка: `go test ./internal/collector/postgres/...`.
- [x] 2.2 `Fetch(ctx, db, sql, params)`: неизвестная база → `unknown database "<name>"`. Дальше `BeginTx(ReadOnly)` → `SET LOCAL statement_timeout` → `Query(sql, pgx.NamedArgs)` → `CollectRows(RowToMap)` → JSON-нормализация → `Rollback` (Commit не вызывается никогда). Пулы спрятаны за интерфейсом с `BeginTx`, чтобы подставлять `pgxmock`. Unit-тесты (pgxmock): порядок вызовов и ReadOnly, rollback при ошибке запроса, неизвестная база, numeric/time после нормализации. Проверка: `go test ./internal/collector/postgres/...`.
- [x] 2.3 Интеграционный тест в том же пакете, гейт `PG_COLLECTOR_TEST_DSN` (без переменной `t.Skip`): `delete`/`create table` падают, `select 1; delete ...` падает, `pg_sleep` упирается в таймаут, `show default_transaction_read_only` = `on`, `@param` со значением `"1; drop table x"` приходит как значение. Проверка: прогон с локальным Postgres, вывод команды приложить; без переменной тест пропускается.

## 3. Встраивание в коллектор и приложение

- [x] 3.1 `internal/collector/collector.go`: интерфейс `SQLFetcher`, поле `pg`, новый аргумент `NewCollector`, ветка `case models.CollectTypePostgres` → `c.pg.Fetch(ctx, crd.CardUUID, crd.Query, crd.Params)`. `go tool mockery`, затем обновить вызовы `NewCollector` в `collector_test.go` и добавить тест ветки pg. Проверка: `go test ./internal/collector/...`.
- [x] 3.2 Конфиг: `Databases map[string]pgcollector.Config \`yaml:"databases"\`` в `internal/config/config.go` с комментарием. В `LogValue` копировать map и маскировать DSN как `postgres://***`. Добавить пример в `config/config.example.yaml` с комментарием «DSN — только read-only пользователь с минимальными правами; dblink и подобные функции обходят read-only транзакцию». Тест: `LogValue` не содержит исходный DSN, а исходный конфиг не изменился. Проверка: `go test ./internal/config/...`.
- [x] 3.3 `internal/app/module_collectors.go`: провайдер pg-коллектора с `fx.Lifecycle` OnStop → `Close()`, передача в `collector.NewCollector`. `cmd/live-server/main.go`: то же самое. Проверка: `go build ./...`, `task run` с одной базой в `databases` стартует, в логе DSN замаскирован.

## 4. Lua

- [x] 4.1 `internal/processor/lua/stdlib/collector.go`: необязательное поле `pg` в `CollectPlugin`. Если оно задано, регистрируется `collector.pg.collect(db, sql, params)` → `rows, nil` | `nil, err`. Подключение в `newLuaStdCollector` (`internal/app/module_report.go`). Тест на мок-fetcher: аргументы доходят как есть, ошибка возвращается вторым значением, без `pg` модуля `collector.pg` нет. Проверка: `go test ./internal/processor/lua/...`.

## 5. Интеграция

- [x] 5.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && golangci-lint run`, вывод приложить. Ручной прогон отчёта с pg-запросом через `task run`: данные попали в датасет.
    - Итог: build OK; `go test ./...` — 17 ok, 3 FAIL, все три падали ещё до изменения (evaluator, pkg/text, processor/duck); `golangci-lint v2.15.0 run --new-from-rev=HEAD` — 0 issues. Ручной прогон через `cmd/live-server` с pg-карточкой: данные с параметром `@from` попали в шаблон, `delete` отклонён (SQLSTATE 25006). Бот (`cmd/bot`) стартует, DSN в логе замаскирован, OnStop закрывает пулы.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/security-review` (изменения в `internal/processor/lua/`, произвольный SQL из Lua).
- `/opsx:archive`, затем коммит `feat(collector): ...`. Миграции `005` и `006` добавить через `git add -f`, потому что `migrations/*` в `.gitignore`.
- Напоминание при деплое: накатить `006_add_query_text.sql` вручную, добавить `databases` в рабочий конфиг.
