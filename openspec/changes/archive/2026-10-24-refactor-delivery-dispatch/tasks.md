# Tasks

## 1. Пакет delivery

- [x] 1.1 Создать `internal/delivery/dispatch.go`: интерфейсы отправителей, `Dispatcher`, `NewDispatcher`, `Send` и помощники — перенос из `internal/models/message.go` без изменения логики; `TgText`. Перенести тесты отправки из `internal/models/message_test.go` в `internal/delivery/dispatch_test.go` без изменения ожиданий. Проверка: `go test ./internal/delivery/` зелёный.
- [x] 1.2 Удалить из `internal/models` методы отправки, интерфейсы отправителей, `send.go`, импорт `internal/delivery/smtp`. Проверка: `go list -deps ./internal/models | grep delivery` пусто; `go test ./internal/models/` зелёный.

## 2. Оркестратор и сборка

- [x] 2.1 `internal/orchestrator`: интерфейс `MessageSender`, `deliver` и `notifyTest` через него; `orchestrator_test.go` на фейк/мок. Проверка: `go test ./internal/orchestrator/` зелёный.
- [x] 2.2 `internal/app/module_delivery.go`, `module_report.go`: провайдер `*delivery.Dispatcher` вместо `*models.SenderProvider`. Проверка: `go build ./...` и `go test ./internal/app/...` (fx-граф, если есть тест).

## 3. Проверка

- [x] 3.1 `CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./... && go tool golangci-lint run` (старые падения `internal/evaluator`, `internal/pkg/text` не в счёт).
- [x] 3.2 `code-simplifier` на изменённых файлах, затем повторить 3.1.

## Workflow follow-up

- `/opsx:archive`, коммит `refactor(delivery): …`.
