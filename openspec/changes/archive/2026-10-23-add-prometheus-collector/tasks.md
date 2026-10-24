# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`.
> Состояние до начала работ (2026-10-23, HEAD `3cc6c73`): `go test ./...` красный в трёх пакетах, не связанных с этим изменением: `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Полный `golangci-lint run` тоже красный (среди прочего 27 `unused`). Из-за этого задача не считается проваленной. Провал — только новые падения или новые замечания линта.
> Сгенерированный код не правим вручную: моки обновляются только через `go tool mockery`.

## 1. Пакет prometheus: конфиг

- [x] 1.1 `internal/collector/prometheus/config.go`: `Config{URL, BearerToken, BasicAuth *BasicAuth{User, Password}, Grafana, DatasourceUID, Timeout}` и `GrafanaConfig{URL, Token, Timeout}` с yaml/json-тегами и комментариями. Дефолт timeout 30s задаётся в коде. Функция сборки источников (`name → baseURL + auth + timeout`) проверяет, что задан ровно один вариант, что grafana-ссылка существует и что `bearer_token` не задан вместе с `basic_auth`. Ошибка содержит имена. Proxy-URL: `<grafana.url>/api/datasources/proxy/uid/<uid>` (D1, D2). Как проверить: табличный тест на валидацию и сборку baseURL и auth проходит, `go test ./internal/collector/prometheus/...`.

## 2. Пакет prometheus: сбор

- [x] 2.1 Подстановка `${name}` (D5) и выбор instant/range с ошибкой неполного range (D4). Сделать функцией, которая строит путь и `url.Values`. Как проверить: unit-тесты на подстановку, неизвестный плейсхолдер, instant с `time` и без, range, неполный range, отсутствие лишних params в API.
- [x] 2.2 `New(cfgs, grafanas, log)` и `Fetch(ctx, target, query, params)`: неизвестный источник → `unknown prometheus source "<name>"`. Дальше GET через клиент с `retry.NewRoundTripper` и декодирование ответа при любом статусе. Результат: `status:error` → текст ошибки, не-JSON → ошибка со статусом, `warnings` → warn в лог, vector/matrix → строки (D6), scalar/string → ошибка (D7). Как проверить: тесты на `httptest.Server` из D10 проходят, `go test ./internal/collector/prometheus/...`.

## 3. Встраивание в коллектор, конфиг и приложение

- [x] 3.1 `internal/collector/collector.go`: переименовать `SQLFetcher` в `QueryFetcher`, добавить поле `prom` и аргумент `NewCollector`, ветку `case models.CollectTypePrometheus`. В `internal/models/report.go` добавить `CollectTypePrometheus = "prom"`. Перегенерировать моки `go tool mockery`, обновить вызовы `NewCollector` в `collector_test.go` и добавить тест ветки prom. Как проверить: `go test ./internal/collector/...`.
- [x] 3.2 `internal/config/config.go`: добавить `Grafana map[string]prometheus.GrafanaConfig \`yaml:"grafana"\`` и `Prometheus map[string]prometheus.Config \`yaml:"prometheus"\`` с комментариями. В `LogValue` копировать map и маскировать `token`, `bearer_token`, `basic_auth.password`. В `config/config.example.yaml` добавить пример из D2 с комментариями: токенам давать минимальные права, переменные Grafana (`$__rate_interval`, `${__range}`) не поддерживаются, вместо них нужны явные params. Как проверить: тест `LogValue` в `config_test.go` (секретов нет, исходный конфиг не изменился), `go test ./internal/config/...`.
- [x] 3.3 `internal/app/module_collectors.go`: провайдер `newPrometheusCollector`, передача в `newCollector`. `cmd/live-server/main.go`: поля `prometheus`/`grafana` в json-конфиге и передача в `NewCollector`. Как проверить: `go build ./...`, `task run` стартует с секцией `prometheus` (секреты в логе замаскированы) и падает на записи, где заданы одновременно `url` и `grafana`.

## 4. Lua

- [x] 4.1 `internal/processor/lua/stdlib/collector.go`: переименовать `SQLCollector` в `QueryCollector`, поле `pg` заменить на `queries map[string]QueryCollector`. Регистрировать `collector.<name>.collect(target, query, params)` для каждого элемента с не-nil значением (`luaPGCollect` обобщить, D8). `internal/app/module_report.go`: передать `{"pg": ..., "prom": ...}`. prom класть в map только при непустом `cfg.Prometheus`, через переменную интерфейсного типа, чтобы не получить typed nil (D8). Как проверить: существующие pg-тесты зелёные, новый тест `collector.prom.collect` на мок (аргументы доходят как есть, ошибка возвращается вторым значением, без prom модуля `collector.prom` нет), `go test ./internal/processor/lua/...`.

## 5. Интеграция

- [x] 5.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Красными допускаются только три пакета из базовой линии. Ручной прогон через `cmd/live-server`: карточка `prom` instant и range с `${...}` против реального или локального Prometheus (`docker run prom/prometheus`) и, если Grafana доступна, через proxy. Данные должны попасть в шаблон.
    - Итог (2026-10-23): build OK; `go test ./...` — 18 ok, 3 FAIL только из базовой линии (evaluator, pkg/text, processor/duck); `golangci-lint run --new-from-rev=HEAD` и отдельно по `internal/collector/prometheus` — 0 issues. Ручной прогон `cmd/live-server` против `prom/prometheus:v2.53.0`: instant с `${job}` и range с `${win}` попали в шаблон, конфиг с `url`+`grafana` роняет старт с именем записи. `task run` бота не запускался (поднимет Telegram и крон с рассылкой). Маскирование секретов проверено тестом `LogValue`. Через proxy Grafana вручную не проверялось (Grafana нет), только `httptest`.

## Workflow follow-up

- После реализации запустить `code-simplifier` на изменённых файлах и повторить проверку.
- Запустить `/security-review`: изменения затрагивают `internal/processor/lua/`, у Lua появится доступ к метрикам.
- Выполнить `/opsx:archive`, затем коммит `feat(collector): коллектор Prometheus (тип prom), напрямую и через Grafana`.
- При деплое добавить в рабочий конфиг секции `grafana` и `prometheus`.
- Следующее изменение — `add-grafana-alerts-collector` (план: /home/netscrawler/.claude/plans/gleaming-brewing-bengio.md, раздел «Изменение 2»).
