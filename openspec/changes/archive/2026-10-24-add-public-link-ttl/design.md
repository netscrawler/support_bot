# Design

## Context

- `public_reports(id, public_id uuid unique, report_id fk → reports on delete cascade)`. Строки вставляются вручную.
- Путь запроса: `handlers.GetGeneratedReportByID` → `service.Report.GenerateReport(ctx, publicID)` → `store.ReportStore.GetByPublicID` (`GetReportIDByPublicID`, затем `GetByID`) → `Orchestrator.Generate`. `models.ErrNotFound` даёт 404.
- `internal/cli/ctl.go` — switch по `os.Args[2]` и `flag.FlagSet` на команду, подключение через `setupDB`. Есть подкоманда с вложенным switch (`script`), её берём за образец.

Решения: grilling 2026-10-23, Q9, Q21, Q27.

## Goals / Non-Goals

**Goals:** срок жизни и отзыв ссылок без SQL, бессрочные ссылки остаются.

**Non-Goals:** TTL по умолчанию, управление ссылками из бота, аудит открытий.

## Decisions

### D1. Миграция
`migrations/init.sql/008_public_report_links_ttl.sql`, добавить в `sqlc.yaml`:
```sql
alter table public_reports
    add column created_at timestamptz not null default now(),
    add column expires_at timestamptz;
```
`expires_at is null` означает бессрочную ссылку, так что существующие строки остаются бессрочными. У старых строк `created_at` получит момент накатки — это допустимо. (агент)

### D2. Отзыв = истечение сейчас (уточнение Q27, подтверждено в Q31)
В Q27 договорились: «отзыв удаляет строку» и «отозванная ссылка отдаёт 410». Эти два пункта противоречат друг другу: после удаления строки отозванную ссылку не отличить от несуществующей, и ответ будет 404. Поэтому `revoke` выполняет `update public_reports set expires_at = now() where public_id = $1 and (expires_at is null or expires_at > now())`. Отдельной колонки `revoked_at` нет: для спеки отзыв и истечение — одно и то же состояние. Если ссылка уже истекла, `revoke` сообщает, что она неактивна, и завершается с кодом 0.

### D3. SQL
В `internal/db/queries/report.sql`:
- `GetReportIDByPublicID` → `GetPublicLink :one`: `select report_id, expires_at from public_reports where public_id = $1`;
- `CreatePublicLink :one`: `insert into public_reports (public_id, report_id, expires_at) values ($1, $2, $3) returning public_id`;
- `ListPublicLinks :many`: join на `reports.name`, необязательный фильтр по имени (`sqlc.narg`), `order by created_at desc`;
- `ExpirePublicLink :execrows` (D2).
Затем `task generate`.

### D4. Проверка срока
`ReportStore.GetByPublicID`: если `expires_at != nil && !expires_at.After(now)`, возвращает `models.ErrLinkExpired` (новая ошибка в `internal/models/errors.go`) до `GetByID`, поэтому генерации нет. Хендлер: `errors.Is(err, models.ErrLinkExpired)` → `httplib.ErrGone` (новая `NewHTTPError(http.StatusGone, "the link has expired")`). Время сравнивает Go (`time.Now()`), а не SQL, чтобы проверку покрывали тесты на `pgxmock`. (агент)

### D5. CLI
`ctl link <create|list|revoke>` по образцу `scriptCtl`: `create` находит `report_id` по имени (существующий `GetAnyReportRowByName`), генерирует `uuid.New()`, `-ttl` разбирает через `flag.Duration` (0 — бессрочно). `list` печатает через `text/tabwriter`. `revoke` при `0 rows` проверяет, существует ли ссылка, и выдаёт «не найдена» (exit 1) или «уже неактивна» (exit 0). Добавить `link` в usage в `Ctl()` и в `help.go`. Логика в `store`/`service`, CLI только разбирает флаги и печатает. (агент)

### D6. Документация
`swagger.yaml`: ответ 410 у `/report/{report_id}`. `README.md`, раздел «HTTP API»: команды `ctl link` и пример ссылки на неделю.

## Risks / Trade-offs

- [Часы БД и бота расходятся] → сравнение только в Go, `now()` в SQL используется только для `created_at` и отзыва. Расхождение в секундах на практике не важно.
- [Ссылки по-прежнему без аутентификации] → вне объёма. Риск снижается сроками и отзывом.

## Migration Plan

1. На существующей БД накатить вручную `008_public_report_links_ttl.sql`.
2. Выкладка релизом. Откат: старый бинарник новые колонки игнорирует, но тогда истёкшие ссылки снова заработают.
