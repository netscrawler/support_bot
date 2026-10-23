# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`. Новый неотслеживаемый пакет линтится отдельно: `go tool golangci-lint run ./internal/collector/zabbix/...`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Сгенерированный код не править: моки — только через `go tool mockery`. Ponytail full.

## 1. Пакет zabbix

- [x] 1.1 `internal/collector/zabbix`: `Config`, `New(cfgs map[string]Config, log) (*Collector, error)` с проверкой url/token (D1), `Fetch(ctx, target, query, params)`. Разбор query и белый список `*.get` до HTTP (D3), подстановка `${name}` в строковые значения (D4), HTTP без retry и ошибки (D5), строки результата через `funcs.Flatten` (D6). В пакете не объявлять интерфейсов. Тесты писать до кода или вместе с ним. Проверка: все тесты пакета из D7 на `httptest.Server` проходят (`go test ./internal/collector/zabbix/...`), линт пакета без замечаний.

## 2. Встраивание

- [x] 2.1 `internal/config/config.go`: поле `Zabbix map[string]zabbix.Config` (yaml `zabbix`, comment) и маскирование токена в `LogValue` с копией map. Тест в `config_test.go`: в логе `***`, исходный map не изменён. Проверка: `go test ./internal/config/...`.
- [x] 2.2 `internal/models/report.go`: `CollectTypeZabbix = "zabbix"`. `internal/collector/collector.go`: поле `zabbix QueryFetcher`, аргумент `NewCollector` после `alerts`, ветка в switch (D2). Обновить вызов `NewCollector` в `collector_test.go`; в `internal/app/module_collectors.go` и `cmd/live-server/main.go` до задачи 2.3 передавать `nil`. Добавить тест ветки `zabbix` в `collector_test.go`. Проверка: `go build ./... && go test ./internal/collector/...`.
- [x] 2.3 `internal/app/module_collectors.go`: провайдер `newZabbixCollector`, который возвращает ошибку `New` (старт падает), с передачей в `newCollector`. `internal/app/module_report.go`: `queries["zabbix"]` только при `len(cfg.Zabbix) > 0` (D2). `cmd/live-server/main.go`: поле `Zabbix` в конфиге (json `zabbix`), создание коллектора с обработкой ошибки, как у prom. Проверка: `go build ./...`.
- [x] 2.4 `config/config.example.yaml`: секция `zabbix` с примером записи. В комментарии: роль пользователя токена (API allow list `*.get`, read-only группы хостов), пример запроса `problem.get` с `${}` и пример `history.get` с `limit` и `time_from`; значения отдаются как есть (коды severity, epoch). Проверка: `task run` или live-server стартует с этой секцией, если подставить реальные url/token; без них парсинг конфига проходит (`go test ./internal/config/...`).

## 3. Интеграция

- [x] 3.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD` и линт нового пакета, вывод приложить. Красные пакеты допустимы только из базовой линии. Ручной прогон через `cmd/live-server` против Zabbix 7.4 (рабочий или локальный `zabbix/zabbix-web-nginx-pgsql:7.4` + server + postgres): карточки `zabbix` с `problem.get` (с `selectHosts`), `host.get` с `countOutput` и `history.get` с `limit` попадают в шаблон; `host.delete` отклоняется без запроса. Результат записать сюда.
    - Итог (2026-10-23): build OK; `go test ./...` красный только в трёх пакетах базовой линии; `golangci-lint --new-from-rev=HEAD` и линт `internal/collector/zabbix` — 0 issues; `config.example.yaml` читается `cleanenv.ReadConfig` в `config.Config`, `zabbix.New` проходит. Ручной прогон `cmd/live-server` против локального Zabbix 7.4.16 (podman: postgres + server + web + agent2, токен через `token.generate`, всегда горящий триггер на `agent.ping`): `problem.get` с `severities: ["${sev}"]` и `selectTags` (колонки `tags.N.tag/value`), `host.get` с `countOutput` (`{result: 1}`), `host.get` с `selectInterfaces` (`interfaces.0.ip`), `history.get` с `limit` попали в шаблон; ошибок в логе нет, токена в логе нет. `host.delete` отклонён ботом (`метод host.delete запрещён`), сбор упал до HTTP. Zabbix принял числовые параметры строками (`history`, `time_from`, `limit`, `severities`), риск из design не подтвердился. Lua-путь `collector.zabbix` проверен тестами механизма `queries` и ревью провайдера, отдельным e2e не проверялся; ограничение роли allow list `*.get` на стенде не проверялось (токен Admin).

## 4. Security review

- [x] 4.1 `/security-review` (2026-10-23): найден обход белого списка `host.delete.get` (Zabbix 7.4 `CJsonRpc.php` берёт только два первых сегмента имени метода). Исправлено: проверка `^[a-z]+\.get$` (design D3), тест на `host.delete.get`, `.get`, `host..get`, `HOST.GET`. Проверка: `go test ./internal/collector/zabbix/...` и полный прогон ниже.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/security-review`: Lua и запросы получают доступ на чтение к Zabbix.
- `/opsx:archive`, затем коммит `feat(collector): коллектор Zabbix (тип zabbix), read-only *.get`.
- При деплое: создать пользователя и роль Zabbix по Migration Plan в design.md, добавить секцию `zabbix` в рабочий конфиг.
