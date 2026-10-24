# Proposal

## Why

Отчёт всегда строится за период, который зашит в `params` запроса, например `=today()` или `=addDays(today(), -7)`. Чтобы получить тот же отчёт за вчера или за прошлый месяц, приходится заводить копию отчёта или править его JSON. Пользователи в боте не могут выбрать период вообще.

## What Changes

- В CEL-выражениях `params` появляются две переменные: `period_from` и `period_to` (timestamp, начало включительно, конец не включительно). Отчёт пишет, например, `"from": "=period_from"`, и коллекторы подставляют значение через уже существующий механизм `params`/`${name}`.
- Четыре периода: `today` (сегодня), `yesterday` (вчера), `week` (7 полных дней до сегодня), `month` (прошлый календарный месяц).
- У отчёта появляется поле `default_period` (по умолчанию `today`). Его используют запуски по cron и «▶️ Запустить».
- Если хотя бы один param отчёта содержит `period_`, кнопка «📤 Получить» в админке и запуск отчёта в ЛК пользователя сначала показывают выбор периода: «Сегодня / Вчера / Неделя / Месяц». У остальных отчётов всё как раньше.
- Тексты запросов (`query`) и отчёты, которые не используют `period_`, не меняются.

## Capabilities

### New Capabilities
- `report-period`: выбор периода данных отчёта при запуске и период по умолчанию для запусков по расписанию.

### Modified Capabilities

## Impact

- Код: `internal/evaluator/evaluate.go` (переменные CEL, сигнатура `EvalStr`), `internal/models/report.go` (`Period`, `DefaultPeriod`, `ResolveParams`, `UsesPeriod`), `internal/generator/generator.go`, `internal/models/event.go` (период в `SpecialEventForLK`), `internal/event_creator/event_api.go`, `internal/orchestrator/orchestrator.go`, `internal/db/queries/report.sql` (+ `task generate`), `internal/store/report_store.go`, `internal/service/report_validator.go`, `internal/tg_bot/{handlers,service}`, `internal/models/report_example.go`, `README.md`.
- БД: миграция `migrations/init.sql/010_report_default_period.sql`, её надо добавить в `sqlc.yaml`. На существующей БД её нужно накатить вручную.
- Зависит от `improve-admin-test-run` (тот же путь «📤 Получить»).
- Вне объёма: произвольные даты и переменные, введённые текстом; период в публичной ссылке; периоды кроме четырёх; часовые пояса, отличные от часового пояса процесса.
