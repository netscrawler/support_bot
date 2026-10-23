# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`. Новый неотслеживаемый пакет линтится отдельно: `go tool golangci-lint run ./internal/collector/grafana/...`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Это не связано с изменением и провалом задачи не считается.
> Сгенерированный код не править: моки — только через `go tool mockery`.

## 1. Пакет grafana

- [x] 1.1 `internal/collector/grafana`: `New(grafanas map[string]prometheus.GrafanaConfig, log)` и `Fetch(ctx, target, mode, params)`. Источники собираются по D1. Проверка режима и ключей params выполняется до запроса (D5). HTTP и ошибки — по D6. Режимы `rules` и `active` разбирают ответ одного endpoint'а (D3), `history` — по D4. Фильтры — по D5. В пакете не объявлять интерфейсов: mockery сгенерирует для них моки. Проверка: все тесты из D7 для пакета на `httptest.Server`, `go test ./internal/collector/grafana/...`, линт пакета без замечаний.

## 2. Встраивание

- [x] 2.1 `internal/models/report.go`: `CollectTypeGrafanaAlerts = "grafana_alerts"`. `internal/collector/collector.go`: поле `alerts QueryFetcher`, аргумент `NewCollector` после `prom`, ветка switch. Обновить вызовы `NewCollector` в `collector_test.go` и добавить тест ветки. Проверка: `go test ./internal/collector/...`.
- [x] 2.2 `internal/app/module_collectors.go`: провайдер `newGrafanaAlertsCollector` с передачей в `newCollector`. `internal/app/module_report.go`: `queries["grafana_alerts"]` только при `len(cfg.Grafana) > 0` (D2). `cmd/live-server/main.go`: создать коллектор из уже имеющегося `cfg.Grafana` и передать в `NewCollector`. `internal/config/config.go`: обновить комментарий секции `grafana`. `config/config.example.yaml`: дополнить комментарий секции `grafana` (используется и для `grafana_alerts`, права токена — Viewer) и добавить пример запроса с режимами, фильтрами и примерами значений state (D5). Проверка: `go build ./...`, `go test ./internal/config/...`.

## 3. Интеграция

- [x] 3.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD` и линт нового пакета, вывод приложить. Красные пакеты — только из базовой линии. Ручной прогон через `cmd/live-server` против локальной Grafana (`podman run grafana/grafana` и правило, которое горит всегда, например на выражении-константе): карточки `grafana_alerts` в режимах `active`, `rules`, `history` попадают в шаблон.
    - Итог (2026-10-23): build OK; `go test ./...` красный только в трёх пакетах базовой линии; `golangci-lint --new-from-rev=HEAD` и линт `internal/collector/grafana` — 0 issues. Ручной прогон `cmd/live-server` против `grafana/grafana:11.2.2` с provisioned правилами и токеном service account (Viewer): `active` (фильтр `state=Alerting`), `rules`, `history` (`from` RFC3339 + `alertname`) попали в шаблон. Прогон выявил и закрыл два расхождения с design: пустой `alertName` в аннотациях (имя берётся из `text`, согласовано) и Normal-экземпляры в `alerts[]` (отбрасываются). Регистрация `collector.grafana_alerts` в Lua проверена тестами механизма `queries` и ревью провайдера, отдельным e2e не проверялась.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/security-review`: Lua получает доступ к алертам.
- `/opsx:archive`, затем коммит `feat(collector): коллектор алертов Grafana (тип grafana_alerts)`.
- При деплое: проверить, что у токена Grafana есть права на чтение правил и аннотаций.
