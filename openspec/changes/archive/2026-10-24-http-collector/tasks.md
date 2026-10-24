# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`. Новый неотслеживаемый пакет линтится отдельно: `go tool golangci-lint run ./internal/collector/httpcoll/...`.
> Сгенерированный код не править: `internal/db/sqlcgen/` — через `internal/db/queries/*.sql` + `task generate`, моки — `go tool mockery`. Тест — до кода или вместе с ним. Ponytail full. Ссылки D* — на design.md.

## 1. Схема и модель

- [x] 1.1 `migrations/init.sql/011_query_options_secrets.sql` (D1) и добавить его в `schema` в `sqlc.yaml`. **На существующей БД миграцию накатить вручную** (`psql -f`). Проверка: `task generate` проходит без ошибок.
- [x] 1.2 `internal/models/report.go`: `CollectTypeHTTP = "http"`, поля `Card.Options`, `Card.Secret` (D2). Проверка: `go build ./...`.
- [x] 1.3 `internal/db/queries/report.sql`: `ListCardsByReportID` и `CreateQuery` с `options`, `secret`; `FindQueryIDByUUIDAndTitle` сравнивает `options` и `secret` (D2); `task generate`. `internal/store/report_store.go`: маппинг обеих колонок, пустые options → `{}`, FK `queries_secret_fkey` → `секрет <name> не найден`. Тесты в `report_store_test.go` (pgxmock): чтение/запись options и secret, ошибка FK. Проверка: `go test ./internal/store/...`.

## 2. Секреты

- [x] 2.1 `internal/pkg/secret/cipher.go`: `NewCipher`, `Encrypt`, `Decrypt` с AAD = имя (D6). Тесты: roundtrip, чужой ключ, чужое имя, ключ не 32 байта, в шифротексте нет открытого значения. Проверка: `go test ./internal/pkg/secret/...`.
- [x] 2.2 `internal/config/config.go`: секция `Secrets.Key` (env `SECRETS_KEY`), проверка длины в `Validate`, маскирование в `LogValue`. Тест в `config_test.go`: ключ 16 байт → ошибка с `SECRETS_KEY`; пустой ключ → ок; в логе `***`. Проверка: `go test ./internal/config/...`.
- [x] 2.3 `internal/db/queries/secret.sql` (`UpsertSecret`, `GetSecret`, `ListSecretNames`, `DeleteSecret`), `task generate`; `internal/store/secret_store.go` (`Set`, `Get` с расшифровкой и `secret.Register`, `List`, `Delete` с FK → «используется запросами», nil `Cipher` → `SECRETS_KEY не задан`). Тесты pgxmock. Проверка: `go test ./internal/store/...`.
- [x] 2.4 `internal/cli/ctl.go`: `ctl secret set -name` (stdin), `list`, `delete -name`; добавить в usage и `help.go`. Значение нигде не печатается. Проверка: `go build ./... && go test ./internal/cli/...`; вручную на локальной БД `echo -n t0k | go run ./cmd/bot ctl secret set -config config/local.yaml -name t && ... list` выводит `t`, в `select value from secrets` нет `t0k`.
- [x] 2.5 Привязка секрета к хосту (D6, spec db-secrets «Секрет привязан к хосту»): `host text not null` в `secrets` в миграции 011 (изменение ещё не заархивировано; локально пересоздать таблицу), `UpsertSecret`/`GetSecret` с host + `task generate`, AAD `name\x00host` в `Cipher`, `SecretStore.Set(name, host, plain)` / `Get → (value, host)`, `ctl secret set -host` (обязателен, `list` печатает `name host`), `httpcoll.SecretGetter` и проверка хоста до запроса. Тесты: cipher (чужой хост в AAD), store, httpcoll (чужой хост — ошибка без запроса, свой хост с портом и без). Проверка: `go build ./... && go test ./... && go tool golangci-lint run`; вручную — секрет на `127.0.0.1` работает, на другой хост — ошибка.

## 3. Коллектор http

- [x] 3.1 `internal/collector/httpcoll/decode.go` + `decode_test.go`: json (root, массив/объект/скаляры, `funcs.Flatten`, нет root), ndjson, csv (BOM, delimiter, несовпадение полей), выбор формата по `options.format`/`Content-Type`, неизвестный тип (D5). Проверка: `go test ./internal/collector/httpcoll/...`.
- [x] 3.2 `internal/collector/httpcoll/{config,collector}.go` + `collector_test.go`: `Config`, `New` с проверкой base_url/auth, `SecretGetter`, `Fetch` и `FetchCard`, разрешение источника, слияние опций (`DisallowUnknownFields`), авторизация, лимит 50 МБ, ошибки не-2xx с обрезкой 512 байт и заменой секрета на `***` (D4). Все сценарии из D9 на `httptest.Server`. Проверка: `go test ./internal/collector/httpcoll/...` и линт пакета без замечаний.
- [x] 3.3 `internal/collector/collector.go`: интерфейс `CardFetcher`, ветка в `collect` (D3). Тест в `collector_test.go`: fetcher с `FetchCard` получает карточку с options/secret, обычный `Fetch` — как раньше. Проверка: `go test ./internal/collector/...`.
- [x] 3.4 Форматы xml, xlsx, yaml, text в `internal/collector/httpcoll/decode.go` (D5a, spec «Формат ответа», «Форма строк из YAML/XML/XLSX и текста»), опция `sheet` в `parseOptions` (валидатор и Lua получают её через `ValidateOptions`/options автоматически), обновить комментарий `http_sources` в `config/config.example.yaml`. Тесты в `decode_test.go` на все сценарии этих требований (xlsx — файл, собранный excelize в тесте). Проверка: `go test ./internal/collector/httpcoll/... ./internal/service/...` и линт пакета.

## 4. Встраивание

- [x] 4.1 `internal/config/config.go`: секция `HTTPSources map[string]httpcoll.Config` (yaml `http_sources`), маскирование `token`/`password` в `LogValue` с копией map, `secret.Register`. Тест в `config_test.go`. Проверка: `go test ./internal/config/...`.
- [x] 4.2 `internal/app/module_collectors.go`: `newHTTPCollector` (с `SecretStore`), `models.CollectTypeHTTP` в `Fetchers`; провайдер `SecretStore` и `Cipher` в fx. `cmd/live-server/main.go`: секция `http_sources`, коллектор с `SecretGetter = nil` (D8a). Проверка: `go build ./... && go test ./internal/app/...` (проверка графа fx).
- [x] 4.3 `internal/service/report_validator.go`: правила для `http` (D8). Тесты: URL без схемы, `ftp://`, неизвестная опция, `format` вне списка, `auth = header` без `auth_header`, валидная карточка. Проверка: `go test ./internal/service/...`.
- [x] 4.4 `internal/processor/lua/stdlib/collector.go` + `internal/app/module_report.go`: `collector.http.collect(target, query, params, options)`, отказ на `secret` (D7). Тест в stdlib: options доходят до `FetchCard`, `secret` → `nil, err` без вызова. Проверка: `go test ./internal/processor/lua/...`.
- [x] 4.5 `config/config.example.yaml` (секция `http_sources`, пример карточки в комментарии, предупреждение про отсутствие allowlist и токены только через `secret`), `config/example.env` (`SECRETS_KEY`, как сгенерировать, что при потере). Проверка: `go test ./internal/config/...` (пример читается).

## 5. Интеграция

- [x] 5.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run`, вывод приложить. Ручной прогон: миграция 011 на локальной БД, `SECRETS_KEY`, `ctl secret set`, `ctl apply` отчёта с тремя `http`-карточками (json+root с bearer из секрета, csv, ndjson) на локальный `python3 -m http.server` / `httptest`-скрипт, `ctl export` (имя секрета есть, значения нет), генерация отчёта через `task run`. Результат записать сюда.
    - Итог (2026-10-24): `go build ./...` OK; `go test ./...` — 43 пакета ok; `golangci-lint run` — 0 issues. Миграция 011 накатана на локальную `bottst`. `SECRETS_KEY` — случайный ключ; `ctl secret set/list` работают, в `secrets.value` нет открытого значения. `ctl apply` отчёта `e2e_http_collector` (неактивный, без cron) с тремя карточками `http` на локальный python-сервер; `ctl export` — `secret: e2e_tok` и `options` на месте, значения секрета нет. Ссылка на несуществующий секрет — `secret "nope" not found`; `ctl secret delete` используемого — `is used by queries`. Сбор через временную программу (`ReportStore.GetByName` → `ResolveParams` → `Collector` с `httpcoll` и `SecretStore`, удалена): `items` (bearer из секрета, `root`, `=period_from` в query) → `{id:1, user.name:a, ...}`, `csv` (`;`, BOM) → `{date, count}`, `nd` → 2 строки. Чужой `SECRETS_KEY` → `decrypt secret "e2e_tok": cipher: message authentication failed`; 401 с токеном в теле → `Bearer ***`, значения в логе нет. Вне scope: `task run` с Telegram не запускался; `ctl apply` с шаблоном `"id": 0` падает на `templates_pkey` — старое поведение; `ctl export` выгружает `params: null` — тоже старое поведение (`RawParams` с `json:"-"`).
    - Повтор после 2.5 и 3.4 (2026-10-24): в локальную `secrets` добавлена колонка `host` (как в обновлённой 011). `ctl secret set` без `-host` → `no secret host provided`; с `-host 127.0.0.1` сохраняется, `list` → `e2e_tok	127.0.0.1`. Отчёт `e2e_http_formats` (8 карточек): json+root+секрет, csv `;`+BOM, ndjson, xml `rss.channel.item` (`{@id, title, creator, tag.0, tag.1}`, префикс `dc:` отброшен), xlsx лист `Data`, yaml `root=items` (`tags.env`), text (`\r` и пустые строки отброшены) — все строки как в spec. Карточка на `localhost:18765` с секретом, привязанным к `127.0.0.1`, → `secret "e2e_tok" is bound to host "127.0.0.1", request host "localhost:18765"`, в логе сервера запроса нет.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/security-review`: Lua-песочница (`internal/processor/lua/`), шифрование секретов, URL из БД.
- `/opsx:archive`, затем коммит `feat(collector): универсальный HTTP-сборщик и секреты в БД`.
- При деплое: накатить `011_query_options_secrets.sql` вручную, задать `SECRETS_KEY`.
