# Tasks

## 1. Реестр коллекторов

- [x] 1.1 `internal/collector/collector.go`: `Fetchers`, `IgnoreQuery`, новый `NewCollector`, lookup в `collect` вместо `switch` (ошибка `unknown fetcher type <type>` для отсутствующего). Обновить `internal/collector/collector_test.go` под новый конструктор и добавить кейс «тип не зарегистрирован → ошибка». Проверка: `go test ./internal/collector/` зелёный.
- [x] 1.2 Перевести `internal/app/module_collectors.go` и `cmd/live-server/main.go` на реестр. Проверка: `go build ./...`.

## 2. Проверка

- [x] 2.1 `CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./... && go tool golangci-lint run` (старые падения `internal/evaluator`, `internal/pkg/text` не в счёт).
- [x] 2.2 `code-simplifier` на изменённых файлах, затем повторить 2.1.

## Workflow follow-up

- `/opsx:archive`, коммит `refactor(collector): …`.
