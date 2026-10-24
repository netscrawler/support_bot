# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Предусловие: `add-report-run-journal` в архиве. Сгенерированный код не править, моки — через `go tool mockery`. Ponytail full.

## 1. Генератор

- [x] 1.1 `internal/models/report.go`: `IgnoreEvaluate bool` с тегом `json:"-"`. `internal/generator/generator.go`: обход условия по D2. Тест в `generator_test.go`: при `IgnoreEvaluate` и ложном условии экспорт есть, `approve=false`; без флага поведение прежнее. Тест сериализации: поле не попадает в JSON отчёта. Проверка: `go test ./internal/generator/... ./internal/models/...`.

## 2. Событие и оркестратор

- [x] 2.1 `models.SpecialEventForLK.Test`, `EventAPI.ProduceTestEvent`, `tg_bot/service.Report.GenerateTestReportByName` (D1). Проверка: `go build ./...`.
- [x] 2.2 `internal/orchestrator/orchestrator.go`: тип `test`, служебные строки и ошибки в чат запросившего, статус и `reason` в журнале (D3). Тесты в `orchestrator_test.go`: ложное условие → отчёт и строка `evaluate = false`, запись `test`/`success`/`evaluate_false`; ошибка генерации → строка `❌` с экранированием, запись `test`/`failed`; `lk` с ложным условием → нет отправки и строк, запись `lk`/`skipped`. Проверка: `go test ./internal/orchestrator/...`.

## 3. Админка

- [x] 3.1 `AdminHandler.GenerateSelectedReport` вызывает `GenerateTestReportByName`, текст ответа по D4. `UserHandler` не трогать. Проверка: `go build ./...`, `go test ./internal/tg_bot/...`.

## 4. Интеграция

- [ ] 4.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Ручной прогон `task run`: «📤 Получить» в админке на отчёте с `evaluate = "[!*]"` → отчёт и строка `evaluate = false`; на отчёте с битым запросом → текст ошибки; в ЛК обычного пользователя на отчёте с `[!*]` ничего не приходит. Результат записать сюда.
  - 2026-10-24 авто: `go build ./...` ок; `go test -count=1 ./...` красный только в базовой линии (`internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`); `golangci-lint --new-from-rev=HEAD` — 0 issues. `go test -race ./internal/orchestrator/...` ок.
  - Ручной прогон: не выполнен (пользователь проверит после всех изменений).

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(tg_bot): пробный запуск отчёта для админа`.
