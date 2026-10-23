# service-observability Specification

## Purpose

Позволяет внешнему мониторингу (docker, systemd, Prometheus) видеть, жив ли бот и как он работает: сколько отчётов прошло и упало, сколько длятся сборы, как часто сбоит доставка.

## Requirements

### Requirement: Отдельный внутренний адрес
Эндпоинты `/metrics` и `/healthz` SHALL обслуживаться отдельным HTTP-сервером на адресе `metrics.addr` (по умолчанию `127.0.0.1:9090`), если `metrics.enabled` равен `true`. По умолчанию `metrics.enabled` равен `false`, и сервер не запускается. Публичный HTTP API MUST NOT отдавать эти эндпоинты.

#### Scenario: Запрос метрик на публичный порт
- **WHEN** клиент запрашивает `/metrics` на порту публичного API
- **THEN** ответ 404

#### Scenario: Сервер отключён
- **WHEN** `metrics.enabled` не задан или равен `false`
- **THEN** бот стартует, внутренний сервер не слушает ни один порт

### Requirement: Healthcheck
`GET /healthz` SHALL отвечать `200` с телом `ok`, если PostgreSQL ответил на ping не дольше чем за 2 секунды, и `503` с текстом ошибки в остальных случаях. Доступность внешних источников (Metabase, Jira, Prometheus и т.д.) и Telegram на ответ MUST NOT влиять.

#### Scenario: База недоступна
- **WHEN** PostgreSQL не отвечает
- **THEN** `/healthz` отвечает 503 не дольше чем через ~2 секунды

#### Scenario: Metabase лежит
- **WHEN** Metabase недоступен, а PostgreSQL отвечает
- **THEN** `/healthz` отвечает 200

### Requirement: Метрики запусков отчётов
`/metrics` SHALL отдавать счётчик запусков отчётов по типу и статусу и гистограмму длительности запусков по типу. Значения SHALL совпадать с записями журнала запусков с момента старта процесса.

#### Scenario: Упавший запуск
- **WHEN** запуск по cron завершился статусом `failed`
- **THEN** `sbot_report_runs_total{kind="cron",status="failed"}` увеличился на 1

### Requirement: Метрики сбора и доставки
`/metrics` SHALL отдавать длительность и число ошибок сбора по типу запроса отчёта (`mb`, `pg`, `prom` и т.д.), а также число повторов и итоговых ошибок отправки по операции канала доставки.

#### Scenario: Ошибка запроса pg
- **WHEN** запрос типа `pg` завершился ошибкой
- **THEN** `sbot_collector_errors_total{type="pg"}` увеличился на 1

#### Scenario: Повтор отправки
- **WHEN** отправка документа в Telegram прошла со второй попытки
- **THEN** `sbot_delivery_retries_total{op="telegram.SendDocument"}` увеличился на 1, `sbot_delivery_failures_total` для этой операции не изменился

### Requirement: Состояние планировщика и версия
`/metrics` SHALL отдавать `sbot_scheduler_running` (1, если cron-задачи запущены, 0, если остановлены из админки) и `sbot_build_info` с версией и коммитом в метках.

#### Scenario: Админ остановил задачи
- **WHEN** админ нажал «остановить задачи» в админке
- **THEN** `sbot_scheduler_running` равен 0, а `/healthz` по-прежнему отвечает 200
