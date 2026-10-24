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
`/metrics` SHALL отдавать счётчик запусков отчётов по типу, статусу и имени отчёта и гистограмму длительности запусков по типу и имени отчёта. Значения SHALL совпадать с записями журнала запусков с момента старта процесса.

#### Scenario: Упавший запуск
- **WHEN** запуск отчёта `daily` по cron завершился статусом `failed`
- **THEN** `sbot_report_runs_total{kind="cron",status="failed",report="daily"}` увеличился на 1

### Requirement: Метрики сбора и доставки
`/metrics` SHALL отдавать длительность и число ошибок сбора по типу запроса (`mb`, `pg`, `prom` и т.д.), имени отчёта и названию карточки, а ошибки — ещё по причине. Гистограмма длительности сбора SHALL различать сборы до 600 секунд. `/metrics` SHALL отдавать число повторов, успешных отправок, длительность и итоговых ошибок (с причиной) по операции канала доставки.

#### Scenario: Ошибка запроса pg
- **WHEN** запрос типа `pg` карточки `orders` отчёта `daily` завершился ошибкой
- **THEN** `sbot_collector_errors_total{type="pg",report="daily",card="orders",reason="other"}` увеличился на 1

#### Scenario: Повтор отправки
- **WHEN** отправка документа в Telegram прошла со второй попытки
- **THEN** `sbot_delivery_retries_total{op="telegram.SendDocument"}` и `sbot_delivery_sent_total{op="telegram.SendDocument"}` увеличились на 1, `sbot_delivery_failures_total` для этой операции не изменился

### Requirement: Состояние планировщика и версия
`/metrics` SHALL отдавать `sbot_scheduler_running` (1, если cron-задачи запущены, 0, если остановлены из админки) и `sbot_build_info` с версией и коммитом в метках.

#### Scenario: Админ остановил задачи
- **WHEN** админ нажал «остановить задачи» в админке
- **THEN** `sbot_scheduler_running` равен 0, а `/healthz` по-прежнему отвечает 200

### Requirement: Причина ошибки в метках
Метка `reason` SHALL принимать только значения `canceled`, `timeout`, `http_4xx`, `http_5xx`, `other`, а для успешного исхода (`status="ok"`) — пустое значение. Текст ошибки MUST NOT попадать в метки.

#### Scenario: Таймаут
- **WHEN** операция завершилась по истечении дедлайна контекста
- **THEN** ошибка учтена с `reason="timeout"`

#### Scenario: Ошибка Telegram API
- **WHEN** Telegram ответил ошибкой с кодом 403
- **THEN** ошибка учтена с `reason="http_4xx"`

### Requirement: Время последнего успешного запуска
`/metrics` SHALL отдавать `sbot_report_last_success_timestamp_seconds{report}` — unix-время завершения последнего запуска отчёта вида `cron` или `manual` со статусом `success` с момента старта процесса. Другие статусы и виды `test`, `lk` значение MUST NOT менять.

#### Scenario: Частичный запуск
- **WHEN** запуск отчёта `daily` завершился статусом `partial`
- **THEN** `sbot_report_last_success_timestamp_seconds{report="daily"}` не изменился

#### Scenario: Тестовый запуск из админки
- **WHEN** тестовый запуск отчёта `daily` завершился статусом `success`
- **THEN** `sbot_report_last_success_timestamp_seconds{report="daily"}` не изменился

### Requirement: Учёт доставки по получателям
`/metrics` SHALL отдавать `sbot_delivery_recipient_total{channel,status,reason}`: каждая попытка доставить сообщение получателю учитывается ровно один раз со `status` `ok` или `failed`, включая ошибки до обращения к каналу. `channel` — `tg`, `max`, `smtp`, `smb` или `unknown` для неподдерживаемого типа получателя.

#### Scenario: Пустой получатель
- **WHEN** у Telegram-получателя не задан чат
- **THEN** `sbot_delivery_recipient_total{channel="tg",status="failed",reason="other"}` увеличился на 1

#### Scenario: Неизвестный тип получателя
- **WHEN** тип получателя не поддерживается
- **THEN** `sbot_delivery_recipient_total{channel="unknown",status="failed",reason="other"}` увеличился на 1

### Requirement: HTTP-повторы к источникам
`/metrics` SHALL отдавать по хосту источника число повторов HTTP-запроса из-за сетевой ошибки (`sbot_source_http_retries_total{host}`) и число запросов, не удавшихся после всех повторов (`sbot_source_http_failures_total{host,reason}`).

#### Scenario: Источник недоступен
- **WHEN** запрос к `metabase.local` падал сетевой ошибкой во всех попытках
- **THEN** `sbot_source_http_retries_total{host="metabase.local"}` увеличился на 3, `sbot_source_http_failures_total{host="metabase.local"}` — на 1

### Requirement: Метрики публичного HTTP API
`/metrics` SHALL отдавать `sbot_http_requests_total{route,code}` и `sbot_http_request_duration_seconds{route}`, где `route` — шаблон маршрута, без значений параметров пути. Запросы, не совпавшие ни с одним маршрутом, MAY не учитываться.

#### Scenario: Запрос отчёта по ссылке
- **WHEN** клиент запросил `/api/v1/public/report/42`, ответ 200
- **THEN** `sbot_http_requests_total{route="GET /api/v1/public/report/{report_id}",code="200"}` увеличился на 1

### Requirement: Метрики Telegram-бота
`/metrics` SHALL отдавать `sbot_bot_updates_total{type,command,status}` и `sbot_bot_update_duration_seconds{type}`. `type` — `message`, `callback` или `other`; `command` — имя известной боту команды или кнопки, иначе `other`; `status` — `ok` или `error` по результату обработки.

#### Scenario: Произвольные данные кнопки
- **WHEN** пришёл callback с данными, не известными боту
- **THEN** учтён с `command="other"`

### Requirement: Метрики пула PostgreSQL
При включённом сервере метрик `/metrics` SHALL отдавать текущее число занятых, простаивающих, всех и максимум соединений пула, а также накопленные число и суммарное время ожидания выдачи соединения и число выдач при пустом пуле.

#### Scenario: Scrape
- **WHEN** Prometheus запрашивает `/metrics`
- **THEN** ответ содержит `sbot_db_pool_acquired_conns` с текущим значением пула

### Requirement: Метрики Lua-скриптов
`/metrics` SHALL отдавать длительность выполнения шага Lua-обработки и число его ошибок (с причиной) по имени скрипта; скрипт, заданный текстом в отчёте, учитывается как `script="inline"`.

#### Scenario: Ошибка скрипта
- **WHEN** скрипт `normalize` вернул ошибку
- **THEN** `sbot_lua_errors_total{script="normalize",reason="other"}` увеличился на 1

### Requirement: Описание метрик
Каждая метрика `sbot_*` SHALL иметь Help-строку на русском, объясняющую смысл. В репозитории SHALL быть `docs/metrics.md` с каждой метрикой `sbot_*`: имя, тип, метки, смысл, пример PromQL.

#### Scenario: Новая метрика
- **WHEN** в `/metrics` есть метрика `sbot_*`
- **THEN** она описана в `docs/metrics.md`
