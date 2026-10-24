# Proposal

## Why

Снаружи нельзя понять, работает ли бот: нет healthcheck для docker и systemd, нет метрик. Если бот завис или лёг вместе с Telegram, уведомление о сбое не дойдёт, и узнать об этом некому. Метрики в Prometheus позволят и алертить на бота снаружи, и строить по нему отчёты им же самим через коллектор `prom`.

## What Changes

- Отдельный внутренний HTTP-сервер (`metrics.enabled`, по умолчанию выключен; `metrics.addr`, по умолчанию `127.0.0.1:9090`) с двумя эндпоинтами:
  - `GET /metrics` — метрики в формате Prometheus;
  - `GET /healthz` — `200 ok`, если PostgreSQL отвечает на ping за 2s, иначе `503` с текстом ошибки.
- Метрики:
  - `sbot_report_runs_total{kind,status}` и `sbot_report_run_duration_seconds{kind}` — по записям журнала запусков;
  - `sbot_collector_duration_seconds{type}` и `sbot_collector_errors_total{type}` — по каждому запросу отчёта;
  - `sbot_delivery_failures_total{op}` (итоговая ошибка после повторов) и `sbot_delivery_retries_total{op}`;
  - `sbot_scheduler_running` (1 или 0);
  - `sbot_build_info{version,commit}`, плюс стандартные `go_*` и `process_*`.
- Публичный API (`http.port`) не меняется, `/metrics` и `/healthz` там не появляются.

## Capabilities

### New Capabilities
- `service-observability`: внутренние эндпоинты состояния и метрик бота.

### Modified Capabilities

## Impact

- Код: новый пакет `internal/metrics` (регистрация метрик и сервер), `internal/runlog` (счётчики запусков), `internal/collector/collector.go`, `internal/pkg/retry/retry.go`, `internal/sheduler/sheduler.go`, `internal/app` (fx-провайдер сервера), `internal/config/config.go` (секция `metrics`), `internal/cli/run.go` (`build_info`), `config/config.example.yaml`.
- Зависимости: `github.com/prometheus/client_golang` уже есть в `go.mod` как indirect (v1.12.1) и становится прямой. Новых модулей нет.
- Зависит от `add-report-run-journal` (метрики запусков) и `sync-delivery-retry` (метрики доставки в `retry.Do`).
- `deploy/ansible` и `docker-compose.yaml` не трогаются. Чтобы метрики были доступны из другого контейнера, адрес задаётся в рабочем конфиге.
- Вне объёма: алерты и дашборды Grafana, проверка внешних источников в healthz, readiness отдельно от liveness.
