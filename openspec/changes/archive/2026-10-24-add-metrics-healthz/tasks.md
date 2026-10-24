# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`. Новый пакет линтится отдельно: `go tool golangci-lint run ./internal/metrics/...`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Предусловия: `add-report-run-journal` и `sync-delivery-retry` в архиве. Сгенерированный код не править. Ponytail full.

## 1. Пакет metrics

- [x] 1.1 `client_golang` как прямая зависимость без смены версии (D6). `internal/metrics`: метрики (D1), `Config`, `Server` с `/metrics` и `/healthz` (D2). Тесты на `httptest`: ping ok → 200 `ok`; ping с ошибкой → 503 и текст; ping дольше 2s → 503; `/metrics` содержит `sbot_build_info`. Проверка: `go test ./internal/metrics/...`, линт пакета, `go mod tidy` без диффа после него.

## 2. Точки записи

- [x] 2.1 Запись метрик в `runlog.Recorder.Record`, `collector.Collector.collect`, `retry.Do`, `sheduler.Sheduler.Start/Stop`, `cli/run.go` (D4). Тесты по разнице `testutil.ToFloat64` до и после: `Record(failed)` → +1 `{cron,failed}`; ошибочный fetch `pg` → +1 `CollectorErrors{pg}`; `Do` успех со 2-й попытки → +1 retries, failures без изменений; `Stop` → gauge 0. Проверка: `go test ./internal/runlog/... ./internal/collector/... ./internal/pkg/retry/... ./internal/sheduler/...`.

## 3. Сборка

- [x] 3.1 `internal/config/config.go`: `Metrics` (D5). fx-провайдер сервера в `internal/app/module_http.go` с пропуском при `metrics.enabled: false` (D2), добавить в `fx.Invoke` в `module.go`. `config/config.example.yaml`: секция `metrics`. Проверка: `go build ./...`, `go test ./internal/config/...`.

## 4. Интеграция

- [x] 4.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Ручной прогон `task run` с `metrics.enabled: true`: `curl 127.0.0.1:9090/healthz` → `ok`; после остановки локального Postgres → 503; `curl 127.0.0.1:9090/metrics | grep sbot_` после одного запуска отчёта показывает `sbot_report_runs_total`; `curl <http.port>/metrics` → 404; «остановить задачи» в админке → `sbot_scheduler_running 0`. Результат записать сюда.
  - Результат 2026-10-24: `go build ./...` ок; линт `--new-from-rev=HEAD` и `./internal/metrics/...` — `0 issues`; `go test ./...` красный в базовой линии (`evaluator`, `pkg/text`, `processor/duck`) и в `collector/appmetrica` — `go vet` на старый `c.log.Debug("parsing response", raw.Status)` в `collector.go:62`, файл не менялся, к изменению не относится.
  - `task run` (копия local.yaml с `metrics.enabled: true` и таймаутом AppMetrica 5s, иначе старт висит 5 минут в `newAppMetricaCollector`): `/healthz` → `ok` 200; `docker stop db` → 503 с текстом ошибки за 3 мс, после `docker start db` → 200; `:8080/metrics` и `:8080/healthz` → 404; `/metrics` содержит `sbot_build_info`, `sbot_scheduler_running 1`, `go_*`; после запроса отчёта по публичной ссылке появился `sbot_collector_duration_seconds{type="mb"}`.
  - Не проверено вживую: `sbot_report_runs_total` (без Telegram не запустить отчёт через журнал; покрыто тестом runlog) и «остановить задачи» → `sbot_scheduler_running 0` (нужна админка в Telegram; покрыто тестом sheduler).

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(metrics): эндпоинты /metrics и /healthz на внутреннем порту`.
- Отдельно, по команде: healthcheck в `docker-compose.yaml`/ansible.
