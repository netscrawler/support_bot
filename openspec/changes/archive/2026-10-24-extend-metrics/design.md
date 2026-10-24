# Design

## Context

Метрики живут в `internal/metrics` (promauto, глобальный реестр) и пишутся из `runlog.Record`, `collector.collect`, `retry.Do`, `sheduler`. Сервер `/metrics` поднимается в `app/module_http.go` только при `metrics.enabled`. Мотивация — proposal.md, требования — specs/service-observability.

Договорённости grilling (2026-10-24): метка `report` на запусках и сборе; все семь областей (доставка, последний успех, HTTP-повторы к источникам, HTTP API, бот, пул БД, Lua); Help на русском + `docs/metrics.md`, без дашборда и алертов, `deploy/` не трогаем; ломать имена/метки можно; `reason` с фиксированным набором; тесты через `prometheus/testutil` для точек с логикой; имя отчёта в сборщик — через `ctx`; на сборе ещё метка `card`; учёт доставки по получателю в `Dispatcher.Send`; DeleteMsg — через `op`, без отдельной метрики; у MAX-бота входящих нет — только Telegram.

## Goals / Non-Goals

**Goals:** найти по метрикам конкретный отчёт, запрос, канал, маршрут, скрипт, который сбоит или тормозит.

**Non-Goals:** трейсинг и время отдельных SQL-запросов; дашборд и алерты; типизация HTTP-ошибок в коллекторах; изменение поведения повторов; метрики MAX-бота.

## Decisions

**D1. `metrics.Reason(err)`** — одна функция, порядок: `context.Canceled` → `canceled`; `context.DeadlineExceeded` или `net.Error.Timeout()` → `timeout`; `*telegoapi.Error` с `ErrorCode` 4xx/5xx → `http_4xx`/`http_5xx`; `*httplib.HTTPError` с `Code` 4xx/5xx → то же; иначе `other`. Коллекторы возвращают нетипизированные `fmt.Errorf("HTTP %d…")`, поэтому их ошибки почти всегда `other` или `timeout` — это фиксируется в `docs/metrics.md`. Альтернатива (типизировать ошибки шести коллекторов) — вне объёма.

**D2. Имя отчёта в ctx.** `metrics.WithReport(ctx, name)` / `metrics.ReportFrom(ctx)` (пустая строка, если нет), по образцу `pkg/logger`. Ставится один раз в `generator.generate` перед `Collect`. Сигнатура `Collect` не меняется. Метка `card` = `crd.Title`.

**D3. Бакеты сбора** — `runBuckets` (0.5…600s) вместо дефолтных (до 10s).

**D4. Доставка в `retry.Do`.** Длительность всей операции с повторами и `sent_total` при успехе по `op`; `failures` с `reason` от исходной ошибки. Unwrap `permanentError`/`afterError` прозрачен для `Reason` (есть `Unwrap`).

**D5. Учёт по получателю в `Dispatcher.Send`.** Один инкремент на получателя после ветки `switch`: `channel` из фиксированной таблицы `models.*Recipient` → `tg|max|smtp|smb`, default → `unknown` (не `r.Type`: значение из конфига). `status` ok/failed, `reason` = `Reason(err)` или пусто при ok.

**D6. HTTP-повторы к источникам** — в `retry.RoundTripper`: `retries{host=req.URL.Host}` перед каждой паузой, `failures{host,reason}` при возврате ошибки после всех попыток или отмене ctx. Поведение не меняется.

**D7. Публичный API** — middleware `Metrics` в `api/http/middlewares`, добавляется в `router.Use(...)` до `Group` (Group копирует список). Статус берётся через существующий `responseRecorder` (по умолчанию 200). `route = r.Pattern`: middleware оборачивает каждый маршрут, поэтому шаблон уже выставлен. Несовпавшие запросы (404 мультиплексора) не учитываются.

**D8. Telegram-бот** — middleware `base.Use(...)` в `tg_bot/router.go` после `PanicRecovery`. `type`: message/callback/other. `command`: для сообщения — `/cmd` из белого списка; для callback — первая запись белого списка, являющаяся префиксом данных (`report_gen…` с кодом периода без разделителя тоже попадает в `report_gen`); иначе `other`. Белый список — литералы предикатов `router.go`, рядом с регистрацией. `status = error`, если `ctx.Next` вернул ошибку. Апдейт без подходящего обработчика проходит через middleware группы `base` (проверяется тестом; если нет — фиксируется здесь).

**D9. Пул БД** — `GaugeFunc`/`CounterFunc` над `pool.Stat()` регистрируются в провайдере сервера метрик (`module_http.go`), где уже есть `db.GetConn()` и пропуск при выключенных метриках; `prometheus.Register` с игнорированием `AlreadyRegisteredError`, чтобы повторная сборка графа fx в тестах не паниковала. Не в `postgres.New` и не через promauto (на init пула нет).

**D10. Lua** — замер в `pipeline.LuaRunner.Run` (знает `ScriptName` и inline-`Script`), а не в `lua.Plugin.Execute` (плагин не знает имени). Ошибки загрузки скрипта (не найден) тоже считаются. `internal/processor/lua/` не меняется — `/security-review` не требуется.

**D11. Последний успех** — `SetToCurrentTime`-аналог `Set(run.FinishedAt.Unix())` в `runlog.Record` только при `success` и только для `cron`/`manual` (как `track` для алертов): тестовый прогон или ЛК не должны маскировать падающий cron-отчёт. Удаления отчёта в процессе бота нет (нет запроса/метода удаления; отчёты меняются в БД), поэтому серии живут до рестарта — это решение Q9 упрощено: удалять нечего.

**D12. Help** — русские строки в `metrics.go`; `docs/metrics.md` — таблица всех `sbot_*` + примеры PromQL (доля ошибок доставки, «давно не было успеха», p95 сбора по карточке).

## Risks / Trade-offs

- [Кардинальность `report×card`] → десятки отчётов × несколько карточек; приемлемо по Q18. Белый список у бота и `route` из шаблона ограничивают остальные метки.
- [Ломаются дашборды на старых метках] → по Q5 потребителей нет; перечислено в proposal.
- [`reason` у коллекторов почти всегда `other`] → честно описано в docs; типизация ошибок — отдельное изменение.
- [Белый список команд бота расходится с роутером] → список рядом с регистрацией в `router.go`; новая кнопка без записи попадёт в `other`, не сломается.

## Migration Plan

Миграций БД нет. После деплоя обновить запросы на старые наборы меток, если появятся.
