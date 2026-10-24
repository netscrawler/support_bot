# Метрики

Prometheus-сервер поднимается на `metrics.addr` (по умолчанию `127.0.0.1:9090`, env `METRICS_ADDR`), только если `metrics.enabled: true` (`METRICS_ENABLED`). Эндпоинты:

- `/metrics` — метрики;
- `/healthz` — `ok` (200) или 503, если не отвечает БД (пинг с таймаутом 2 с).

Кроме `sbot_*` экспортируются стандартные `go_*` и `process_*`.

## Запуски отчётов

Пишутся в `runlog.Record` при записи каждого запуска.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_report_runs_total` | counter | `kind` (cron/manual/lk/test), `status` (success/skipped/failed/partial), `report` | Запуски отчётов |
| `sbot_report_run_duration_seconds` | histogram | `kind`, `report` | Длительность запуска (`FinishedAt - StartedAt`) |
| `sbot_report_last_success_timestamp_seconds` | gauge | `report` | Unix-время последнего запуска `cron`/`manual` со статусом `success` (`test` и `lk` не учитываются) |

## Сбор данных

Пишутся в `collector.collect`: замер вокруг `Fetch` одной карточки. Имя отчёта берётся из `ctx` (ставит `generator.generate`); `card` — заголовок карточки.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_collector_duration_seconds` | histogram | `type`, `report`, `card` | Длительность сбора карточки (успешного и нет) |
| `sbot_collector_errors_total` | counter | `type`, `report`, `card`, `reason` | Ошибки сбора, включая неизвестный тип источника |

## Доставка

Операция `op` — имя вызова в `retry.Do`, например `smtp.Send`, `smb.Upload`, `telegram.DeleteMsg`.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_delivery_sent_total` | counter | `op` | Операции, завершившиеся успехом (после всех повторов) |
| `sbot_delivery_failures_total` | counter | `op`, `reason` | Операции, завершившиеся ошибкой: исчерпаны повторы или ошибка permanent. Отмена ctx не считается |
| `sbot_delivery_retries_total` | counter | `op` | Принятые решения о повторе (инкремент перед паузой) |
| `sbot_delivery_duration_seconds` | histogram | `op` | Длительность всей операции с повторами |
| `sbot_delivery_recipient_total` | counter | `channel` (tg/max/smtp/smb/unknown), `status` (ok/failed), `reason` | Итог по каждому получателю в `Dispatcher.Send`; при `ok` `reason` пустая |

## HTTP-запросы к источникам

Пишутся в `retry.RoundTripper`. 3 повтора, паузы 15 с, 45 с, 135 с.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_source_http_retries_total` | counter | `host` | Принятые решения о повторе; инкремент перед паузой, поэтому отмена ctx во время паузы повтор всё равно учитывает |
| `sbot_source_http_failures_total` | counter | `host`, `reason` | Запрос не удался после всех попыток или прерван отменой ctx |

## Публичный API

Middleware `Metrics` в `api/http/middlewares`. `route` — шаблон маршрута (`r.Pattern`), не URL.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_http_requests_total` | counter | `route`, `code` | Запросы по маршруту и коду ответа (по умолчанию 200) |
| `sbot_http_request_duration_seconds` | histogram | `route` | Длительность обработки (бакеты по умолчанию) |

## Telegram-бот

Middleware в `tg_bot/router.go`, после `PanicRecovery`. Только Telegram.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_bot_updates_total` | counter | `type` (message/callback/other), `command`, `status` (ok/error) | Обработанные апдейты. `command`: команда из `botCommands` (`start`, `help`, `admin`, `add`) или префикс данных из `botCallbacks`; иначе `other` |
| `sbot_bot_update_duration_seconds` | histogram | `type` | Длительность обработки апдейта |

## Пул БД

Регистрируются в `module_http.go` вместе с сервером метрик, значения читаются из `pgxpool.Stat()` при скрейпе.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_db_pool_acquired_conns` | gauge | — | Занятые соединения |
| `sbot_db_pool_idle_conns` | gauge | — | Простаивающие соединения |
| `sbot_db_pool_total_conns` | gauge | — | Всего соединений |
| `sbot_db_pool_max_conns` | gauge | — | Максимальный размер пула |
| `sbot_db_pool_acquire_total` | counter | — | Успешные получения соединения |
| `sbot_db_pool_empty_acquire_total` | counter | — | Получения при пустом пуле (с ожиданием) |
| `sbot_db_pool_acquire_wait_seconds_total` | counter | — | Суммарное время получения соединения, секунды |

## Lua

Пишутся в `pipeline.LuaRunner.Run`. `script` — `script_name` шага или `inline` для встроенного скрипта. Ошибка загрузки скрипта (не найден) тоже считается.

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_lua_duration_seconds` | histogram | `script` | Длительность шага (загрузка + выполнение) |
| `sbot_lua_errors_total` | counter | `script`, `reason` | Ошибки шага |

## Служебные

| Метрика | Тип | Метки | Что значит |
|---|---|---|---|
| `sbot_scheduler_running` | gauge | — | 1, пока планировщик запущен; 0 после `Stop` |
| `sbot_build_info` | gauge | `version`, `commit` | Всегда 1; версия и коммит сборки (`cli/run.go`) |

## Метка `reason`

Функция `metrics.Reason(err)`, порядок проверок:

| Значение | Условие |
|---|---|
| `canceled` | `errors.Is(err, context.Canceled)` |
| `timeout` | `context.DeadlineExceeded` или `net.Error.Timeout()` |
| `http_4xx` / `http_5xx` | `*telegoapi.Error` (по `ErrorCode`) или `*httplib.HTTPError` (по `Code`) |
| `other` | всё остальное |

`http_*` сегодня реально бывает только у ошибок Telegram API и у `httplib.HTTPError`. Коллекторы возвращают нетипизированные ошибки (`fmt.Errorf("HTTP %d…")`), поэтому в `sbot_collector_errors_total` почти всегда `other` или `timeout`; по HTTP-коду источника их не разделить.

## Особенности

- Серии с `report`/`card` живут до рестарта процесса: удаление или переименование отчёта их не убирает.
- Не учитываются: запросы, не совпавшие с маршрутом (404 мультиплексора), и паники в HTTP-обработчиках.
- Неизвестные команды и callback'и бота дают `command="other"`. Апдейты, не подошедшие ни под один обработчик, считаются со `status="ok"`.
- `channel="unknown"` — неподдерживаемый тип получателя.
- У MAX-бота входящих апдейтов нет, метрик бота для него нет.
- Метрики пула есть только при включённом сервере метрик.
- `sbot_delivery_failures_total` и `sbot_delivery_recipient_total` считают разное: первая — операции `retry.Do`, вторая — получателей.

## Примеры PromQL

Доля неудачных доставок по каналу за 1 час:

```promql
sum by (channel) (increase(sbot_delivery_recipient_total{status="failed"}[1h]))
/
sum by (channel) (increase(sbot_delivery_recipient_total[1h]))
```

Отчёты без успеха больше 26 часов:

```promql
time() - sbot_report_last_success_timestamp_seconds > 26*3600
```

p95 длительности сбора по карточке:

```promql
histogram_quantile(0.95,
  sum by (le, report, card) (rate(sbot_collector_duration_seconds_bucket[1h])))
```

Топ ошибок сбора по отчёту за сутки:

```promql
topk(10, sum by (report) (increase(sbot_collector_errors_total[24h])))
```

RPS и p95 публичного API по маршруту:

```promql
sum by (route) (rate(sbot_http_requests_total[5m]))

histogram_quantile(0.95,
  sum by (le, route) (rate(sbot_http_request_duration_seconds_bucket[5m])))
```

Среднее ожидание соединения пула БД (секунды на получение):

```promql
rate(sbot_db_pool_acquire_wait_seconds_total[5m]) / rate(sbot_db_pool_acquire_total[5m])
```
