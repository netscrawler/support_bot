# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Сгенерированный код не править: `internal/db/sqlcgen/` — через `task generate`, моки — через `go tool mockery`. Ponytail full.

## 1. Схема и хранилище

- [x] 1.1 `migrations/init.sql/008_public_report_links_ttl.sql` (D1), добавить в `sqlc.yaml`. Запросы из D3, `task generate`. Проверка: `task generate` без ошибок, `go build ./...`.
- [x] 1.2 `models.ErrLinkExpired`, проверка срока в `ReportStore.GetByPublicID` (D4), методы store `CreatePublicLink`, `ListPublicLinks`, `ExpirePublicLink`. Тесты на `pgxmock`: `expires_at` null → отчёт; в прошлом → `ErrLinkExpired` без запроса `GetByID`; в будущем → отчёт. Проверка: `go test ./internal/store/...`.

## 2. HTTP

- [x] 2.1 `httplib.ErrGone`, ветка 410 в `GetGeneratedReportByID` (D4). Тест хендлера: `ErrLinkExpired` → 410, `ErrNotFound` → 404. `swagger.yaml`: ответ 410 (D6). Проверка: `go test ./internal/api/... ./internal/pkg/httplib/...`.

## 3. CLI

- [x] 3.1 `ctl link create|list|revoke` (D5), usage и `help.go`, раздел в `README.md` (D6). Тест разбора аргументов и вывода `list` (без БД, через интерфейс store или фейк). Проверка: `go test ./internal/cli/...`, `sbot ctl link` без аргументов печатает usage.

## 4. Интеграция

- [x] 4.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Ручной прогон на локальной БД с накатанной 008: `ctl link create -report <имя> -ttl 1m` → ссылка отдаёт файл, через минуту 410; `ctl link create` без ttl → работает; `ctl link revoke` → 410; `ctl link list` показывает состояния; старая ссылка, вставленная до миграции, работает. Результат записать сюда.
  - Результат 2026-10-24: `go build ./...` ок; `go test ./...` красный только в базовой линии (`evaluator`, `pkg/text`, `processor/duck`); линт `0 issues`.
  - Локальная БД: накатаны 005, 007, 008, 009. `ctl link create` с `-ttl 1m` и без него печатает UUID; несуществующий отчёт → rc=1. `revoke` → «ссылка отозвана», повторно → «уже неактивна» (rc=0), неизвестный UUID → «ссылка не найдена» (rc=1). `list` показывает «истекла» для отозванной и `-`/«активна» для бессрочных, в том числе для строки, вставленной SQL без `expires_at` (как старые ссылки).
  - HTTP 410/404 вживую не проверялись: старт бота висит на `newAppMetricaCollector` (сетевой вызов, не связан с изменением), по решению пользователя не ждали. Покрыто тестами store (`ErrLinkExpired` без `GetByID`) и хендлера (410/404).

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(api): срок действия и отзыв публичных ссылок на отчёты`.
- При деплое: накатить вручную `008_public_report_links_ttl.sql`.
