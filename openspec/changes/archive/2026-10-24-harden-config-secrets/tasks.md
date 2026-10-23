# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`. Новый пакет линтится отдельно: `go tool golangci-lint run ./internal/pkg/secret/...`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Предусловие: `add-report-run-journal` в архиве. Сгенерированный код не править. Ponytail full.

## 1. Тип секрета

- [x] 1.1 `internal/pkg/secret`: `String` (D1), `Register`, `Redact` (D3). Тесты: структура с вложенным `String` в `slog` JSON и text handler, `fmt.Sprintf("%v %+v %#v")`, `json.Marshal` → нет исходного значения; `Redact` с вложенными секретами (длинный раньше короткого), значения короче 4 не регистрируются. Проверка: `go test ./internal/pkg/secret/...`, линт пакета.
- [x] 1.2 Перевести все поля из Context design на `secret.String`, места использования — на `.Value()` (D2), в том числе `cmd/live-server`. По каждому полю grep его использований на `Sprintf|Errorf|Fprintf|Sprint`, в каждом месте `.Value()`, список проверенных мест приложить сюда. Тесты на настоящее значение: `postgres.Config.getDSN()` содержит реальный пароль; для jira, appmetrica, grafana, prometheus (bearer и basic auth), zabbix — через `httptest.Server` реальный токен доходит до заголовка запроса. Упростить или удалить `Config.LogValue`. Тесты в `internal/config`: лог конфига со всеми секретами не содержит значений (по каждому полю и по значениям map); `config update` (`marshalConfig`) сохраняет реальные значения; тест-рефлексия из Risks. Проверка: `go build ./...`, `go test ./internal/config/... ./internal/cli/...`.
  - Проверенные места (2026-10-24), везде `.Value()`: `max_bot/bot.go:30` (NewApi), `tg_bot/bot.go:42` (NewBot), `collector/jira/collector.go:70` (Bearer), `collector/appmetrica/collector.go:120` (OAuth), `collector/zabbix/collector.go:54`, `collector/grafana/collector.go:49`, `collector/prometheus/config.go:79,81,98` (bearer, basic auth, grafana), `collector/postgres/config.go:39` (ParseConfig), `postgres/config.go:55` (UserPassword), `postgres/database.go:21` (ParseConfig от `getDSN().Value()`), `delivery/smtp/client.go:51` (PlainAuth), `delivery/smb/reconn.go:48`. Сравнения с `""` оставлены без `Value()`. grep `Sprintf|Errorf|Fprintf|Sprint` по этим полям — пусто; контрольный grep использований без `.Value()` — только присваивание DSN в `postgres/config.go`.
  - Попутно: `pkg/generate_env.go` (`config create -format env`) брал значение через `fmt.Sprint` и писал бы `***` — для `Kind()==String` теперь `fv.String()`. `collector/appmetrica/collector.go:62` — исправлен старый `go vet` (slog без ключа). `Config.LogValue` и `safeConfig` удалены. `cmd/live-server` не менялся; его образец `configDef()` теперь печатает `"***"` вместо плейсхолдеров `token`/`oauth_token`.

## 2. Redact в журнале

- [x] 2.1 `config.Load`: `secret.Register` всех секретов, включая пароль из DSN (D3). `internal/runlog`: `Redact` текста ошибки до записи и до уведомления. Тест в `runlog`: ошибка с зарегистрированным токеном → в записи и уведомлении `***`. `internal/pkg/retry`: в warn-логе повтора ошибка через `secret.Redact(err.Error())`; тест: зарегистрированный токен в тексте ошибки не попадает в лог (slog в буфер). Проверка: `go test ./internal/runlog/... ./internal/config/... ./internal/pkg/retry/...`.

## 3. Validate

- [x] 3.1 `Config.Validate()` по D4 с путями по yaml-тегам. Табличные тесты: минимальный корректный конфиг проходит; по одному тесту на каждое правило; две ошибки разом. Проверка: `go test ./internal/config/...`, `sbot config validate` на `config/config.example.yaml` проходит (или пример дополняется, чтобы проходил).

## 4. Интеграция

- [x] 4.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Ручной прогон: `task run` с `config/local.yaml` → в строке `starting with config` нет ни одного секрета (`grep` по значениям из local.yaml в выводе пуст); конфиг без токена Telegram → старт падает с понятной ошибкой. Результат записать сюда.
  - Результат 2026-10-24: `go build ./...` ок; `go test ./...` красный только в базовой линии (`evaluator`, `pkg/text`, `processor/duck`; `collector/appmetrica` теперь зелёный — исправлен vet); линт `0 issues`.
  - `task run` с копией `local.yaml`: в строке `starting with config` все секреты `***` (`Password:***`, `DSN:***`, `Token:***`). grep 9 значений полей `token|password|dsn|secret` из local.yaml по всему логу старта: совпадения только ложные — пароль локальной БД равен имени пользователя `postgres` (совпадает с `User:postgres`, `user=postgres`), и `refresh_token_ttl: 720h` не секрет.
  - Без `telegram_token`: бот завершается с кодом 1 и `validate config: telegram.telegram_token: не задано`; `sbot config validate` на таком конфиге — код 1, на `config/config.example.yaml` — код 0.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(config): тип secret.String и полная проверка конфига при старте`.
- Перед деплоем: `sbot config validate` на рабочем конфиге.
