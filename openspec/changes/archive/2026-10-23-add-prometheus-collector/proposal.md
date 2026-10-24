# Proposal

## Why

Для отчётов нужны метрики из стека Grafana/Prometheus, а сейчас бот берёт данные только из Metabase, AppMetrica, Jira и Postgres. Prometheus не всегда доступен из сети бота напрямую, иногда к нему можно попасть только через Grafana. Поэтому нужны оба пути.

## What Changes

- Новый тип запроса `prom`. В `card_uuid` хранится имя Prometheus-источника из конфига, PromQL лежит в `queries.query`, в `params` передаются время запроса и значения для подстановки.
- Если в params есть `start`/`end`/`step`, выполняется range-запрос, иначе instant (с необязательным `time`).
- В PromQL подставляются плейсхолдеры `${name}` из params, неизвестный плейсхолдер даёт ошибку.
- Результат превращается в плоские строки: labels становятся колонками, `__name__` переименовывается в `metric`, добавляются `value` (число или null) и для range-запроса `timestamp` (RFC3339).
- Новая секция конфига `grafana`: имя → `url`, `token`, `timeout`. Её же будет использовать будущий коллектор алертов.
- Новая секция конфига `prometheus`: имя → либо прямой `url` (с необязательными `bearer_token` или `basic_auth`), либо `grafana` + `datasource_uid` для доступа через datasource proxy Grafana. Запись, где не задан ровно один из вариантов, считается ошибкой конфига при старте. Секреты маскируются в логах.
- Коллектор доступен из Lua как `collector.prom.collect(name, promql, params)`.

## Capabilities

### New Capabilities
- `prometheus-collector`: сбор данных отчёта PromQL-запросом к Prometheus-совместимому API, напрямую или через Grafana, в том числе из Lua-скриптов.

### Modified Capabilities

## Impact

- Код: новый пакет `internal/collector/prometheus`, ветка `prom` в `internal/collector/collector.go`, константа в `internal/models/report.go`, `internal/config/config.go` (секции и маскирование), `internal/app/module_collectors.go`, `internal/app/module_report.go`, `internal/processor/lua/stdlib/collector.go`, `cmd/live-server/main.go`, `config/config.example.yaml`.
- БД: без изменений, колонка `queries.query` уже есть.
- Безопасность: Lua-скрипты получат доступ на чтение к метрикам всех Prometheus-источников из конфига. Перед archive обязателен `/security-review`.
- Зависимости: новых нет, используются `net/http` и `encoding/json`.
- Вне объёма (отдельное изменение `add-grafana-alerts-collector`): алерты Grafana. Чтение готовых панелей дашбордов Grafana — на будущее.
