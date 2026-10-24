# Proposal

## Why

Публичная ссылка `GET /api/v1/public/report/{uuid}` работает без аутентификации и вечно: при каждом открытии она заново генерирует отчёт по свежим данным. Ссылки создаются вставкой строки в `public_reports` руками в БД. Отозвать утёкшую ссылку тоже можно только через SQL. Ограничить срок жизни ссылки нельзя совсем.

## What Changes

- У ссылки появляется необязательный срок действия `expires_at`. Без срока ссылка вечная, как сейчас. Существующие ссылки остаются вечными.
- Истёкшая или отозванная ссылка отвечает `410 Gone`. Несуществующая — `404`, как и раньше.
- Новые команды CLI:
  - `sbot ctl link create -config <cfg> -report <name> [-ttl 168h]` создаёт ссылку и печатает UUID и путь;
  - `sbot ctl link list -config <cfg> [-report <name>]` выводит ссылки с отчётом, датой создания, сроком и состоянием;
  - `sbot ctl link revoke -config <cfg> <uuid>` отзывает ссылку.
- В `swagger.yaml` добавляется ответ 410.

## Capabilities

### New Capabilities
- `public-report-links`: публичные ссылки на генерацию отчёта, их срок действия, создание и отзыв.

### Modified Capabilities

## Impact

- Код: `internal/db/queries/report.sql` (+ `task generate`), `internal/store/report_store.go`, `internal/models/errors.go` (ошибка «ссылка истекла»), `internal/service/report_generator.go`, `internal/api/http/handlers/get_public_generated_report_by_id.go`, `internal/pkg/httplib/errors.go` (`ErrGone`), `internal/cli/ctl.go` (+ `help.go`), `swagger.yaml`, `README.md` (раздел HTTP API).
- БД: миграция `migrations/init.sql/008_public_report_links_ttl.sql` (колонки `created_at`, `expires_at`), её надо добавить в `sqlc.yaml`. На существующей БД её нужно накатить вручную.
- Вне объёма: создание и отзыв ссылок из Telegram-бота, TTL по умолчанию на уровне отчёта или конфига, аутентификация публичного API, учёт открытий ссылки.
