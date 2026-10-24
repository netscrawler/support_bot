# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Предусловия: `add-report-run-journal` и `improve-admin-test-run` в архиве. Сгенерированный код не править: `internal/db/sqlcgen/` — через `task generate`, моки — через `go tool mockery`. Ponytail full.

## 1. Схема и модель

- [x] 1.1 `migrations/init.sql/009_report_dedup.sql` (D1), добавить в `sqlc.yaml`. `r.dedup` во все select и в `CreateReport`, `data_hash` в `InsertReportRun`, новый запрос `LastReportRunSuccessHash`, `task generate`. Проверка: `task generate`, `go build ./...`.
- [x] 1.2 `models.Report.Dedup`, маппинг в `report_store.go`, `report_example.go` (D3). `models.Dataset.Hash()` (D2). Тесты: хэш не зависит от порядка ключей и от `_meta`, зависит от порядка строк и значений; JSON без `dedup` → false; `export`/`apply` сохраняют флаг (тест store на `pgxmock`). Проверка: `go test ./internal/models/... ./internal/store/...`.
  - Отступление: `report_example.go` не менялся. `ctl create` выводит шаблон через `json.MarshalIndent`, а `omitempty` скрывает `false`. Флаг описан в README.

## 2. Оркестратор

- [x] 2.1 `RunRecorder.LastSuccessHash` в `runlog` и `run_store`, `.mockery.yaml` не меняется (интерфейс уже в списке), `go tool mockery`. Подавление в `generateAndDeliver` (D4). Тесты в `orchestrator_test.go`: `dedup`+`cron`+тот же хэш → нет отправки, `skipped`/`duplicate`; другой хэш → отправка; нет прошлого → отправка; `manual` с тем же хэшем → отправка; `dedup=false` → отправка; ошибка `LastSuccessHash` → отправка. Проверка: `go test ./internal/orchestrator/... ./internal/runlog/...`.

## 3. Документация

- [x] 3.1 `README.md`, раздел про отчёты: флаг `dedup`, что сравнивается, совет класть изменчивое время в `_meta`, пример «алерты раз в 5 минут». Проверка: раздел есть, пример JSON проходит `sbot ctl validate`.

## 4. Интеграция

- [ ] 4.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Ручной прогон `task run` на БД с 007–009: отчёт с `dedup: true` на cron `*/1 * * * *` и статичным запросом отправляется один раз, дальше `skipped`/`duplicate` в «🧾 Запуски»; при смене данных снова отправляется. Результат записать сюда.
  - 2026-10-24 авто: `go build ./...` ок; `go test -count=1 ./...` красный только в базовой линии (`internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`); `golangci-lint --new-from-rev=HEAD` — 0 issues. Пример из README проходит `sbot ctl validate`.
  - Ручной прогон: не выполнен (пользователь проверит после всех изменений).

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(orchestrator): дедупликация отправки отчёта с неизменившимися данными`.
- При деплое: накатить вручную `009_report_dedup.sql`.
