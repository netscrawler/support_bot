# Proposal

## Why

Метрики бота сейчас отвечают на вопрос «что-то сломалось», но не на вопрос «что именно»: в метках нет имени отчёта и запроса, у доставки нет успешных отправок (не посчитать долю ошибок), ошибки доставки вне повторов не учитываются, а HTTP API, Telegram-бот, пул PostgreSQL, Lua и HTTP-повторы к источникам не видны вовсе. Смысл метрик нигде не описан.

## What Changes

- **BREAKING** `sbot_report_runs_total`, `sbot_report_run_duration_seconds` получают метку `report`.
- **BREAKING** `sbot_collector_duration_seconds` и `sbot_collector_errors_total` получают метки `report` и `card`, ошибки — ещё `reason`; бакеты длительности сбора как у запусков (до 600s).
- **BREAKING** `sbot_delivery_failures_total` получает метку `reason`.
- Новое: `sbot_report_last_success_timestamp_seconds{report}`.
- Новое: `sbot_delivery_sent_total{op}`, `sbot_delivery_duration_seconds{op}`.
- Новое: `sbot_delivery_recipient_total{channel,status,reason}` — каждая доставка получателю ровно один раз, включая ошибки до вызова канала.
- Новое: `sbot_source_http_retries_total{host}`, `sbot_source_http_failures_total{host,reason}`.
- Новое: `sbot_http_requests_total{route,code}`, `sbot_http_request_duration_seconds{route}` для публичного API.
- Новое: `sbot_bot_updates_total{type,command,status}`, `sbot_bot_update_duration_seconds{type}` для Telegram-бота.
- Новое: метрики пула PostgreSQL `sbot_db_pool_*`.
- Новое: `sbot_lua_duration_seconds{script}`, `sbot_lua_errors_total{script,reason}`.
- Подробные Help-строки на русском и `docs/metrics.md` с таблицей метрик и примерами PromQL.

## Capabilities

### New Capabilities

_нет_

### Modified Capabilities

- `service-observability`: метки `report`/`card`/`reason` у существующих метрик; новые метрики доставки, последнего успеха, HTTP-повторов к источникам, публичного API, Telegram-бота, пула БД и Lua; документация метрик.

## Impact

- Код: `internal/metrics`, `internal/runlog`, `internal/collector`, `internal/generator`, `internal/pkg/retry`, `internal/delivery/dispatch.go`, `internal/api/http`, `internal/tg_bot`, `internal/processor/pipeline`, `internal/app/module_http.go`.
- Документация: `docs/metrics.md`.
- Внешние потребители метрик: дашборды и алерты по старым наборам меток надо обновить (внешних потребителей, по словам владельца, нет).
- Зависимости не добавляются.
