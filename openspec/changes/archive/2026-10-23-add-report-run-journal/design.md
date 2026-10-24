# Design

## Context

Путь запуска сейчас такой: `Orchestrator.run` → `processGenReportEvent(ctx, event, active, recipientOverride)` → горутина `generateAndDeliver` → `generator.Generate` (возвращает `dataset, data, approve, err`) → `deliver` → `models.Message.Send`. Ошибки на каждом шаге только пишутся в лог.

- Тип запуска виден по тому, откуда пришло событие: `EventC` с типом gen — cron; `SpecialEventC` с `EventTypeGenReport` — кнопка «▶️ Запустить» (`ProduceGenEvent`); `SpecialEventC` с `EventTypeGenReportForTG` — отправка в чат запросившего (`ProduceSpecialEvent`, кнопки «📤 Получить» в админке и в ЛК).
- `Message.Send` перебирает получателей и склеивает ошибки через `errors.Join`, но не сообщает, сколько получателей отчёт получили. Без этого не отличить `partial` от `failed`.
- Уведомление админов уже есть: `tg_bot/service.Notify.SendAdminNotify` (через `ListAdmins`, роли admin и primary). `telegram.ChatAdaptor.SendText` шлёт с `ParseMode: HTML`.
- Генерация по публичной ссылке идёт через `Orchestrator.Generate`, а не через `generateAndDeliver`.

Решения приняты в grilling 2026-10-23 (Q3, Q4, Q13–Q15, Q17, Q29). Решения агента помечены «(агент)».

## Goals / Non-Goals

**Goals:**
- Одна точка, где запуск превращается в запись журнала и при необходимости в уведомление.
- Минимальная правка оркестратора: он вычисляет итог, а запись и уведомления делает отдельный компонент.

**Non-Goals:**
- Журнал генераций по публичной ссылке (`Orchestrator.Generate`).
- Учёт доставки на уровне каждого получателя. Хранится только итог и склеенная ошибка.
- Уведомления в Max и на email.

## Decisions

### D1. Таблица `report_runs` (Q13)
Миграция `migrations/init.sql/007_create_report_runs.sql`, добавить её в `schema` в `sqlc.yaml`:
```sql
create table report_runs (
    id          bigserial primary key,
    report_name text        not null,
    kind        text        not null, -- cron | manual | lk | test
    status      text        not null, -- success | skipped | failed | partial
    reason      text,                 -- для skipped: evaluate (позже duplicate)
    error       text,
    chat_id     bigint,               -- для lk/test: чат запросившего
    started_at  timestamptz not null,
    finished_at timestamptz not null
);
create index report_runs_name_started on report_runs (report_name, started_at desc);
create index report_runs_started on report_runs (started_at);
```
Связи по `report_name`, а не FK на `reports.id`, потому что `LoadByEvent` отдаёт `models.Report` без id, а запись журнала не должна пропадать при пересоздании отчёта через `ctl apply`. (агент)
`error` хранится обрезанным до 4000 символов, чтобы зацикленная ошибка не раздувала таблицу. (агент)

### D2. Модель и тип запуска
- `models.Run{ReportName, Kind, Status, Reason, Error string; ChatID *int64; StartedAt, FinishedAt time.Time}` и константы `RunKind*`, `RunStatus*` в `internal/models/run.go`.
- `processGenReportEvent` получает параметр `kind models.RunKind` вместо того, чтобы выводить его по `active` и `recipientOverride`. Вызовы в `run`: `EventC` gen → `cron`, `EventTypeGenReport` → `manual`, `EventTypeGenReportForTG` → `lk`. Тип `test` появляется в `improve-admin-test-run`. Для `lk` `ChatID` берётся из `recipientOverride.Chat.ChatID`.

### D3. `Message.Send` сообщает число доставок
Сигнатура меняется на `Send(ctx, sp) (sent []SentMessage, delivered int, err error)`, где `delivered` — число получателей без ошибки. Тогда `err != nil && delivered > 0` даёт `partial`, `err != nil && delivered == 0` даёт `failed`. Вызов один, в `orchestrator.deliver`. Альтернатива — возвращать ошибки по каждому получателю — отвергнута: даёт больше кода, а журналу нужен только итог.

### D4. Где вычисляется итог
`generateAndDeliver` засекает `started_at`, получает итог через `generate` и `deliver` (который теперь возвращает `delivered` и ошибку отправки, а не глотает её) и в конце один раз вызывает `o.runs.Record(ctx, run)`. Интерфейс `RunRecorder{ Record(ctx, models.Run) }` объявляется в `internal/orchestrator` и добавляется в `.mockery.yaml` (`go tool mockery`). Пустой список получателей даёт `failed` с ошибкой `empty targets list`.

Пустой экспорт: `generator.generate` логирует ошибку каждого `exporter.Export` и идёт дальше, поэтому может вернуть `approve=true` и пустой `res`. Тогда `NewMessage` пустой, `sendTg` ничего не шлёт и возвращает nil, и журнал записал бы `success` для запуска, который ничего не доставил. Поэтому генератор собирает ошибки экспорта через `errors.Join` и возвращает их как `err`, если `len(res) == 0 && len(report.Exports) > 0`. Оркестратор пишет `failed` с этой ошибкой. Если упала часть экспортов, отчёт уходит с остальными, как сейчас, а ошибки только логируются. Отчёт без экспортов вообще (`len(report.Exports) == 0`) по-прежнему не ошибка. Ошибка `SaveTgMsg` статус не меняет, как и сейчас (warn в лог).

### D5. Пакет `internal/runlog` (Q14, Q17)
`Recorder` с зависимостями: `Store` (`LastStatus(ctx, report) (status, ok, err)` только по `kind in ('cron','manual')`, `Insert`, `DeleteOlderThan`), `Alerter`, конфиг.
`Record`:
1. Если тип `cron` или `manual`, читает прошлый статус (`LastStatus`). `test` и `lk` в состоянии уведомлений не участвуют (Q29): упавший ЛК-запуск пользователя, который заблокировал бота, не должен ни будить админов, ни глушить следующий настоящий сбой по cron.
2. `Insert(run)`.
3. Только для `cron` и `manual`: при текущем `failed`/`partial` и прошлом не `failed`/`partial` (или отсутствии прошлого) отправляет уведомление о сбое. При текущем `success`/`skipped` и прошлом `failed`/`partial` отправляет уведомление «снова работает».
4. Любая ошибка (store, alert) только пишется в лог. `Record` ничего не возвращает и не паникует.

Гонка двух одновременных запусков одного отчёта может дать дубль уведомления. Принимаем: семафор оркестратора и так редко запускает один отчёт параллельно. (агент)

### D6. Уведомление (Q3, Q15)
`Alerter` — интерфейс в `runlog` с двумя реализациями-функциями внутри одной структуры: если `alerts.chat_id != 0`, используется `telegram.ChatAdaptor.SendText(ctx, models.TgChat{ChatID, ThreadID}, msg)`, иначе `service.Notify.SendAdminNotify(ctx, msg)`. Текст:
```
⚠️ Отчёт <b>{name}</b> ({kind}): {status}
<pre>{html.EscapeString(error[:500])}…</pre>
```
и `✅ Отчёт <b>{name}</b> снова работает ({kind})`. Экранирование обязательно, потому что `SendText` шлёт с HTML и `<` в ошибке ломает отправку. Обрезка по рунам, а не по байтам. (агент)

### D7. Очистка (Q4)
`Recorder.Start(ctx)`: сразу и затем каждые 24 часа (`time.Ticker`) вызывает `DeleteOlderThan(now - runs.retention)`. Запускается в `appCtx`, как generator и orchestrator, и останавливается вместе с ним. Привязка к deleter отвергнута: deleter срабатывает только по cron с типом delete, а он может быть не настроен.

### D8. Конфиг
В `internal/config/config.go`:
- `Alerts runlog.AlertsConfig` (`yaml:"alerts"`): `ChatID int64` (`chat_id`, env `ALERTS_CHAT_ID`), `ThreadID int` (`thread_id`).
- `Runs runlog.Config` (`yaml:"runs"`): `Retention time.Duration` (`retention`, `env-default:"720h"`).

Секретов в них нет, `LogValue` не меняется. Плюс образец в `config/config.example.yaml`.

### D9. SQL и store
`internal/db/queries/report_run.sql`: `InsertReportRun :exec`, `LastReportRunStatus :one` (`where report_name=$1 and kind in ('cron','manual') order by started_at desc limit 1`), `ListReportRuns :many` (`... order by started_at desc limit $2`), `DeleteReportRunsBefore :execrows`. Затем `task generate`. `internal/store/run_store.go` по образцу `sent_msg_store.go`, тест на `pgxmock`.

### D10. Кнопка в админке
- `getMarkupForReport` (`internal/tg_bot/handlers/helpers.go`): вторая строка с кнопкой `🧾 Запуски`, callback `report_runs;{id};{name};{page}`. Префикс не пересекается с `report_get` и `report_resend` при `CallbackDataContains`.
- `AdminHandler.ListRuns` берёт `service.Report.ListRuns(ctx, name, 10)` (новый метод, store через существующий `ReportProvider`/repo сервиса) и рендерит строки `02.01 15:04 · cron · ✅ success · 3.2s · <ошибка 200 символов>`. Кнопка «Назад» возвращает в карточку отчёта.

## Risks / Trade-offs

- [В журнал не попадает генерация по публичной ссылке] → это осознанно (Q13), она не доставляет отчёт получателям.
- [При Telegram-сбое уведомление о сбое Telegram тоже не дойдёт] → запись в журнале и лог остаются. Мониторинг снаружи появится в `add-metrics-healthz`.
- [Фоновые повторы Telegram/Max (`internal/pkg/retry`) могут доставить сообщение после того, как запуск записан как `failed`] → ложное уведомление. Исправляется в `sync-delivery-retry`, его стоит делать сразу после этого изменения.
- [`Message.Send` меняет сигнатуру] → один вызов, тесты в `internal/models` и `internal/orchestrator` обновляются в той же задаче.

## Migration Plan

1. На существующей БД накатить вручную `migrations/init.sql/007_create_report_runs.sql`.
2. Добавить в рабочий конфиг `alerts.chat_id` (необязательно) и `runs.retention` (необязательно).
3. Откат: старый бинарник таблицу не использует, её можно оставить или удалить через `drop table report_runs`.
