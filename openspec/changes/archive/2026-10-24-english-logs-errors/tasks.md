# Tasks

## 1. Ошибки коллекторов

- [x] 1.1 Перевести ошибки в `internal/collector/grafana/collector.go`, `internal/collector/prometheus/{collector,config,query}.go`, `internal/collector/zabbix/collector.go`; обновить `grafana_test.go:263`, `zabbix_test.go:98,246,249`. Проверка: `go test ./internal/collector/...` зелёный.

## 2. Остальные ошибки

- [x] 2.1 Перевести ошибки в `internal/config/config.go:161`, `internal/exporter/xlsx/exporter.go`, `internal/cli/link.go:188,200`; обновить `internal/cli/link_test.go:146`. Проверка: `go test ./internal/config/... ./internal/exporter/... ./internal/cli/...` зелёный.

## 3. Логи

- [x] 3.1 Перевести сообщения логов в `internal/orchestrator/orchestrator.go:326,339` и `internal/runlog/runlog.go:73,98,141,160,195,197` (алерты в Telegram на строках 120 и 129 не трогать). Проверка: `go test ./internal/orchestrator/... ./internal/runlog/...` зелёный.

## 4. Сообщения тестов

- [x] 4.1 Перевести кириллицу в `t.Fatal`/`t.Errorf`/`t.Skip`/`assert.True` и тестовые `errors.New` в `internal/delivery/smb/client_test.go`, `internal/models/{dataset,report}_test.go`, `internal/metrics/pool_test.go`, `internal/config/config_test.go:127`, `internal/collector/postgres/integration_test.go:19`, `internal/orchestrator/deleter_test.go:84,94`, `internal/runlog/runlog_test.go:227`. Проверка: `go test ./...` зелёный.

## 5. Проверка

- [x] 5.1 AST-скан кириллических литералов в аргументах вызовов: остались только строки из Non-Goals `design.md`. Затем `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run` — зелёный.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, повторная проверка.
- `/opsx:archive`, коммит `refactor(logs): логи и ошибки на английском`.
