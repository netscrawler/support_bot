# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`. Новый неотслеживаемый пакет линтится отдельно: `go tool golangci-lint run ./internal/runlog/...`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Сгенерированный код не править: `internal/db/sqlcgen/` — через `task generate`, моки — через `go tool mockery`. Ponytail full.

## 1. Схема и хранилище

- [x] 1.1 `migrations/init.sql/007_create_report_runs.sql` по D1, добавить в `schema` в `sqlc.yaml`. `internal/db/queries/report_run.sql` с запросами из D9, `task generate`. Проверка: `task generate` без ошибок, `go build ./...`.
- [x] 1.2 `internal/models/run.go`: `Run`, `RunKind*`, `RunStatus*` (D2). `internal/store/run_store.go`: `Insert`, `LastStatus` (только `cron`/`manual`), `List(name, limit)`, `DeleteOlderThan`. Ошибка обрезается до 4000 символов. Тесты на `pgxmock` писать до кода или вместе с ним: обрезка, `LastStatus` без строк даёт `ok=false`. Проверка: `go test ./internal/store/...`.

## 2. Итог запуска в оркестраторе

- [x] 2.1 `internal/models/message.go`: `Send` возвращает `delivered` (D3). Тест в `internal/models`: два получателя, один падает → `delivered=1`, ошибка не nil. Проверка: `go test ./internal/models/...`.
- [x] 2.2 `internal/generator/generator.go`: ошибка, если все экспорты упали (D4, «Пустой экспорт»). Тест в `generator_test.go`: два экспорта, оба падают → `err` с обеими ошибками; один из двух падает → `res` из одного, `err == nil`; ноль экспортов → `err == nil`. `internal/orchestrator/orchestrator.go`: интерфейс `RunRecorder`, параметр `kind` в `processGenReportEvent` (D2), вычисление статуса и один вызов `Record` в `generateAndDeliver` (D4). `.mockery.yaml` + `go tool mockery`. Тесты в `orchestrator_test.go` на каждый статус (`success`, `skipped`/`evaluate`, `failed` по ошибке генерации, `failed` при всех упавших экспортах, `failed` без получателей, `partial`) и на типы `cron`/`manual`/`lk` с `ChatID`. Проверка: `go test ./internal/orchestrator/...`.

## 3. Запись, уведомления, очистка

- [x] 3.1 Пакет `internal/runlog`: `Recorder.Record` по D5, `Alerter` по D6 (HTML-экранирование, обрезка 500 по рунам), `Start` с очисткой по D7, `Config`/`AlertsConfig` (D8). Тесты: первый `failed` → одно уведомление; второй `failed` подряд → нет; `success` после `failed` → «снова работает»; `test` и `lk` → нет уведомлений и `LastStatus` не читается; `cron` success → `lk` failed → `cron` failed → уведомление о сбое есть; ошибка `Insert` не паникует; `<` в ошибке экранируется. Проверка: `go test ./internal/runlog/...` и линт пакета.
- [x] 3.2 `internal/config/config.go`: поля `Alerts`, `Runs` (D8). `internal/app`: провайдеры `Recorder`, `RunStore`, `Alerter` (выбор по `alerts.chat_id`), передача в `orchestrator.New`, запуск `Recorder.Start` в `appCtx`. `config/config.example.yaml`: секции `alerts` и `runs` с комментариями. Проверка: `go build ./...`, `go test ./internal/config/...`.

## 4. Админка

- [x] 4.1 `internal/tg_bot`: кнопка `🧾 Запуски` и обработчик `ListRuns` с роутингом `report_runs` (D10), метод сервиса `ListRuns`. Тест форматирования строки запуска (статус, длительность, обрезка ошибки до 200). Проверка: `go test ./internal/tg_bot/...`.

## 5. Интеграция

- [x] 5.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD` и линт `internal/runlog`, вывод приложить. Ручной прогон `task run` на локальной БД с накатанной 007: отчёт с ошибочным запросом по cron раз в минуту → одно уведомление за три запуска; после исправления запроса → «снова работает»; кнопка «🧾 Запуски» показывает записи. Результат записать сюда.
  - 2026-10-23: build ок; `go test ./...` красный только в базовой линии (`evaluator`, `pkg/text`, `processor/duck`); линт `--new-from-rev=HEAD` и `internal/runlog`, `internal/store` — 0 issues. Ручной прогон `task run` по решению пользователя не делался — выполнить при деплое.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(runlog): журнал запусков отчётов и уведомления о сбоях`.
- При деплое: накатить вручную `007_create_report_runs.sql`, задать `alerts.chat_id` в рабочем конфиге.
