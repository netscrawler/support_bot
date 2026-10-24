# Tasks

## 1. Частичный результат Load

- [x] 1.1 Тест в `internal/store/report_store_test.go` (pgxmock): два отчёта, у одного `ListCardsByReportID` падает → `Load` возвращает второй отчёт и ошибку с именем первого. Подтвердить падение на текущем коде, затем исправить `Load`; doc-комментарий у `service.ReportStore.Load`.
- [x] 1.2 `internal/cli/ctl.go` export: сохранить собранные отчёты, затем вернуть ошибку `Load`, если она была. Проверка: `go build ./...`, `go vet ./internal/cli/`.

## 2. Тип строки и мёртвый код

- [x] 2.1 Ввести `reportRow`, `assemble(ctx, reportRow)`, перевести шесть вызовов на литералы с именованными полями. Проверка: `go test ./internal/store/` зелёный.
- [x] 2.2 Удалить `ReportStore.LoadActive` и `LoadActive` из `orchestrator.ReportLoader`, перегенерировать мок `go tool mockery`. Проверка: `go build ./... && go test ./internal/orchestrator/` зелёные.
- [x] 2.3 Удалить ставший неиспользуемым запрос `ListActiveReportRows` из `internal/db/queries/report.sql`, `go tool sqlc generate`. Проверка: `go build ./...`.

## 3. Проверка

- [x] 3.1 `CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./... && go tool golangci-lint run` (старые падения `internal/evaluator`, `internal/pkg/text` не в счёт).
- [x] 3.2 `code-simplifier` на изменённых файлах, затем повторить 3.1.

## Workflow follow-up

- `/opsx:archive`, коммит `refactor(store): …`.
