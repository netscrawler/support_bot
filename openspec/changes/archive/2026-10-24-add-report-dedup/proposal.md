# Proposal

## Why

Отправку «по событию» сейчас делают так: частый cron плюс условие `evaluate` (например, «есть горящие алерты»). Пока условие истинно, отчёт с теми же данными уходит на каждом срабатывании cron, например каждые 5 минут, и забивает чат. Нужна защита: не отправлять повторно то, что уже отправлено.

## What Changes

- Новый флаг отчёта `dedup` (по умолчанию выключен). Задаётся в JSON отчёта (`ctl apply`/`export`).
- Если флаг включён, запуск по cron с теми же данными, что и последний успешный запуск, не отправляется. В журнале он получает статус `skipped` с причиной `duplicate`.
- «Те же данные» означает совпадение хэша датасета после pipeline, без листа `_meta`.
- Ручной запуск (`manual`), пробный (`test`) и из ЛК (`lk`) отправляются всегда. Хэш успешного ручного запуска тоже считается «последним отправленным».
- Хэш хранится в журнале запусков, в новой колонке `report_runs.data_hash`.

## Capabilities

### New Capabilities
- `report-dedup`: подавление повторной отправки отчёта с неизменившимися данными.

### Modified Capabilities

## Impact

- Код: `internal/models/report.go` (поле `Dedup`), `internal/db/queries/report.sql` и `report_run.sql` (+ `task generate`), `internal/store/report_store.go`, `internal/store/run_store.go`, `internal/models/run.go` (`DataHash`), новая функция хэша датасета в `internal/models/dataset.go`, `internal/orchestrator/orchestrator.go`, `internal/runlog`, `internal/service/report_validator.go` (если там перечисляются поля), `internal/models/report_example.go` (`ctl create`).
- БД: миграция `migrations/init.sql/009_report_dedup.sql` (`reports.dedup`, `report_runs.data_hash`), её надо добавить в `sqlc.yaml`. На существующей БД её нужно накатить вручную.
- Зависит от `add-report-run-journal` и `improve-admin-test-run`.
- Вне объёма: дедупликация по отдельным строкам («новые алерты с прошлой отправки»), окно времени («не чаще раза в час»), дедупликация для каждого получателя.
