# Tasks

## 1. Секреты в ошибке пробного запуска

- [x] 1.1 Тест в `internal/orchestrator/orchestrator_test.go`: секрет зарегистрирован через `secret.Register`, генерация падает с ошибкой, содержащей секрет → служебное сообщение админу содержит `***`, а не секрет. Подтвердить падение на текущем коде, затем `errText` вызывает `secret.Redact` до обрезки; тест зелёный.

## 2. Нейтральный skipped/duplicate

- [x] 2.1 Константа причины `duplicate` в `internal/models` (рядом с `RunStatus*`), использовать в `orchestrator.go`.
- [x] 2.2 Тест `internal/runlog/runlog_test.go`: success → failed → skipped/duplicate (нет уведомления) → success (одно «снова работает»); `fakeStore.LastStatus` пропускает дубли, как SQL. Подтвердить падение, затем `Record` не уведомляет на дубль; тест зелёный.
- [x] 2.3 `internal/db/queries/report_run.sql` `LastReportRunStatus`: `and (status <> 'skipped' or reason is distinct from 'duplicate')`, затем `task generate`; в `git diff internal/db/sqlcgen/` меняется только этот запрос. Обновить ожидание текста запроса в `internal/store/run_store_test.go`, если он сверяется; `go test ./internal/store/` зелёный.

## 3. Ошибки Grafana

- [x] 3.1 Тесты в `internal/collector/grafana/grafana_test.go`: rules со `status: error` → ошибка содержит HTTP-код и начало тела; history с невалидным JSON → ошибка содержит код и начало тела. Подтвердить падение, затем исправить `ruleRows`/`historyRows`; тесты зелёные.

## 4. Проверка

- [x] 4.1 `CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./... && go tool golangci-lint run` (известные старые падения `internal/evaluator` и `internal/pkg/text` не в счёт).
- [x] 4.2 `code-simplifier` на изменённых файлах, затем повторить 4.1.

## Workflow follow-up

- `/opsx:archive`, коммит `fix(runlog): …`.
