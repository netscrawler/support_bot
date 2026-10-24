# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`.
> Базовая линия (2026-10-23): `go test ./...` на HEAD 55f0604 (2026-10-24) зелёный полностью. Провал из базовой линии задачу не валит — сверить с прогоном на HEAD до изменения.
> Сгенерированный код не править. Ponytail full. Миграций БД нет.

## 1. Основа metrics

- [x] 1.1 `internal/metrics`: `Reason(err)` (D1), `WithReport`/`ReportFrom` (D2), новые метрики и метки из proposal, `CollectorDuration` на `runBuckets` (D3), все Help на русском (D12). Табличный тест `Reason`: nil-обёртки, `context.Canceled`, `DeadlineExceeded`, `net.Error` с Timeout, `*telegoapi.Error` 403/502, `*httplib.HTTPError` 404, обёрнутая через `fmt.Errorf("%w")`, прочая → `other`. Тест `ReportFrom` без значения → `""`. Проверка: `go test ./internal/metrics/...`; `go build ./...` пока может быть красным из-за смены меток — допустимо только внутри задач 1–2, чинится в 2.1.

## 2. Запуски и сбор

- [x] 2.1 `runlog.Record`: метка `report`, `ReportLastSuccess` только при `success` (D11). `generator.generate`: `metrics.WithReport` перед `Collect` (D2). `collector.collect`: метки `type,report,card`, ошибки с `reason`. Обновить `runlog_test.go` и `collector_test.go` под новые метки; добавить: `Record(success)` выставляет timestamp = `FinishedAt`, `Record(partial)` не меняет; ошибка fetch с `WithReport(ctx,"daily")` → +1 `{pg,daily,<card>,other}`. Проверка: `go build ./... && go test ./internal/runlog/... ./internal/collector/... ./internal/generator/...`.

## 3. Доставка

- [x] 3.1 `retry.Do`: `DeliveryDuration{op}`, `DeliverySent{op}` при успехе, `DeliveryFailures{op,reason}` (D4). Обновить `retry_test.go`; добавить: успех со 2-й попытки → sent +1; Permanent с `context.DeadlineExceeded` внутри → failures `{op,timeout}` +1. Проверка: `go test ./internal/pkg/retry/...`.
- [x] 3.2 `Dispatcher.Send`: `DeliveryRecipient{channel,status,reason}` по каждому получателю, default → `unknown` (D5). Тесты в `dispatch_test.go`: tg без чата → `{tg,failed,other}`; неизвестный тип → `{unknown,failed,other}`; успешная smtp → `{smtp,ok,""}`. Проверка: `go test ./internal/delivery/...`.
- [x] 3.3 `retry.RoundTripper`: `SourceHTTPRetries{host}`, `SourceHTTPFailures{host,reason}` (D6). Тест с фейковым транспортом, всегда возвращающим ошибку, и ctx, отменяемым после первой паузы (паузы 15s не ждать): retries +1, failures `{host,canceled}` +1; успешный ответ → счётчики не меняются. Проверка: `go test ./internal/pkg/retry/...`.

## 4. Входящие: HTTP API и бот

- [x] 4.1 Middleware `Metrics` в `internal/api/http/middlewares` + в `router.Use(...)` в `server.go` (D7). Тест через `httplib.Router` с маршрутом `GET /report/{id}`: запрос `/report/42` → `sbot_http_requests_total{route="GET /report/{id}",code="200"}` +1; хэндлер, пишущий 500 → `code="500"`. Проверка: `go test ./internal/api/...`.
- [x] 4.2 Middleware метрик Telegram-бота в `internal/tg_bot` + `base.Use(...)` в `router.go`, белый список из литералов предикатов (D8). Тест функции разметки: `/start` → `start`; `/foo` → `other`; callback `report_gen_today` → `report_gen`; `admin_report_page;3` → `admin_report_page`; `zzz` → `other`. Тест middleware: `ctx.Next` с ошибкой → `status="error"`; апдейт без подходящего обработчика проходит через `base`-middleware (иначе записать в design D8). Проверка: `go test ./internal/tg_bot/...`.

## 5. Пул БД и Lua

- [x] 5.1 Метрики пула над `pool.Stat()` в провайдере сервера метрик `app/module_http.go`, регистрация с игнорированием `AlreadyRegisteredError` (D9). Тест в `internal/metrics`: функция регистрации, вызванная дважды, не паникует, `/metrics` содержит `sbot_db_pool_acquired_conns` (stat-функция подменяемая, без реальной БД). Проверка: `go test ./internal/metrics/... ./internal/app/...`.
- [x] 5.2 `pipeline.LuaRunner.Run`: `LuaDuration{script}`, `LuaErrors{script,reason}`, inline → `script="inline"` (D10). Тест: inline-скрипт с `error()` → `{inline,other}` +1. Проверка: `go test ./internal/processor/pipeline/...`.

## 6. Документация и интеграция

- [x] 6.1 `docs/metrics.md`: каждая `sbot_*` (имя, тип, метки, смысл), значения `reason` и где реально бывает `http_*` (D1), примеры PromQL из D12. Проверка: скрипт сравнения — `grep -o 'Name: *"sbot_[a-z_]*"' internal/metrics/*.go` против имён в docs, расхождений нет.
- [x] 6.2 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run`, вывод приложить; `curl /metrics` на локальном запуске с `metrics.enabled: true` — есть `sbot_db_pool_*`, `sbot_bot_*` после команды боту, `sbot_http_*` после запроса к API. Результат записать сюда.
  - Результат 2026-10-24: build ок, `go test ./...` — все пакеты `ok`, линт — `0 issues`.
  - Живой запуск (копия local.yaml с `metrics.enabled: true`, `addr 127.0.0.1:9091`, таймаут AppMetrica 5s): `/healthz` → `ok`; `/metrics` содержит `sbot_db_pool_*` (acquire_total 6, max_conns 10), `sbot_source_http_retries_total{host="api.appmetrica.yandex.ru"} 1` и `sbot_source_http_failures_total{…,reason="timeout"} 1`, `sbot_scheduler_running 1`; после `GET :8080/api/v1/public/report/<uuid>` (404 от хэндлера) — `sbot_http_requests_total{code="404",route="GET /api/v1/public/report/{report_id}"} 1`; `:8080/metrics` → 404.
  - Не проверено вживую: `sbot_bot_*` (нужна команда в Telegram; покрыто тестом `tg_bot/metrics_test.go` на реальном `th.BotHandler`), `sbot_report_*`/`sbot_delivery_*`/`sbot_lua_*` (нужен запуск отчёта; покрыты юнит-тестами).

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(metrics): детальные метрики отчётов, доставки, API, бота, БД и Lua`.
