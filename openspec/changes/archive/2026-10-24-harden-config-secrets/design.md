# Design

## Context

- `internal/cli/run.go` вызывает `cfg.Validate()`, затем пишет `slog.Any("config", cfg)`. `Config.LogValue()` подменяет 8 полей и копирует 4 map-секции с маскированием.
- Важная особенность slog: `LogValuer` срабатывает только на верхнем уровне значения. Вложенные структуры JSON-хендлер сериализует через `encoding/json`, а текстовый — через `fmt`. Поэтому одного `LogValue()` у типа поля мало: нужны ещё `MarshalJSON` и `String`/`GoString`.
- `config create/update` строят YAML через `pkg.StructToYAMLNode(cfg)` и `yaml.Marshal`.
- Поля-секреты (найдено grep по `config.go`): `postgres.Config.Password`, `.DSN`; `tgbot.Config.TelegramToken`; `maxbot.Config.Token`; `smtp.Config.Password`; `smb.Config.Password`; `jira.Config.AuthToken`; `appmetrica.Config.OAuthToken`; `pgcollector.Config.DSN`; `prometheus.Config.BearerToken`, `BasicAuth.Password`; `prometheus.GrafanaConfig.Token`; `zabbix.Config.Token`.

Решения: grilling 2026-10-23, Q10, Q22, Q28.

## Goals / Non-Goals

**Goals:** маскирование задаётся типом поля, а не списком; полный список ошибок конфига при старте.

**Non-Goals:** маскирование секретов, которые пришли не из конфига (например, токен внутри данных отчёта или Lua-скрипта).

## Decisions

### D1. Пакет `internal/pkg/secret`
```go
type String string
func (s String) Value() string                { return string(s) }
func (String) String() string                 { return "***" }
func (String) GoString() string               { return "***" }
func (String) LogValue() slog.Value           { return slog.StringValue("***") }
func (String) MarshalJSON() ([]byte, error)   { return []byte(`"***"`), nil }
```
`MarshalYAML` не реализуется, поэтому `StructToYAMLNode`/`yaml.Marshal` пишут реальное значение (это проверяется тестом `config update`). Для чтения: базовый тип `string`, поэтому cleanenv, yaml и `json.Unmarshal` (live-server) читают без доработок. Если `StructToYAMLNode` разбирает поля по `reflect.Kind`, а не по типу, — он уже работает. Иначе его надо научить работать с `Kind() == String`. Проверяется в задаче 1.2. (агент)

Пустой `String` тоже печатается как `***`. Так нельзя отличить «не задан» от «задан» в логе, но в этом случае проверка конфига (D4) и так падает. (агент)

### D2. Перевод полей
Каждое поле из списка в Context получает тип `secret.String`. Места использования переходят на `.Value()`. Компилятор находит только часть из них: передачу в параметр `string` (`telego.NewBot`, `smtp.PlainAuth`, `url.UserPassword`, `Header.Set("…"+token)`). А `fmt.Sprintf("%s"/"%v")`, `Errorf`, `Sprint` и `Fprintf` на `secret.String` **компилируются** и тихо подставляют `***`, так что битый DSN или заголовок `Authorization` проявится только в рантайме. Поэтому для каждого поля обязателен grep его использований по `Sprintf|Errorf|Fprintf|Sprint`, и в каждом месте должен стоять `.Value()`. Плюс тесты из задачи 1.2: реальный секрет доходит до DSN и до HTTP-заголовков. `postgres.Config.getDSN()` возвращает `secret.String`, а `pgxpool` получает `.Value()`. В `Config.LogValue()` удаляется маскирование полей и копирование map. Метод остаётся только как `slog.AnyValue(safeConfig(c))`, а если без него вывод не меняется, удаляется целиком. (агент)

### D3. `Redact`
`secret.Register(values ...String)` складывает непустые значения в пакетный список под `sync.RWMutex`. `secret.Redact(s string) string` заменяет каждое значение на `***`: сначала длинные, чтобы короткий секрет внутри длинного не оставил хвост. Значения короче 4 символов не регистрируются, чтобы не испортить весь текст. `config.Load` после чтения вызывает `Register` для всех секретов, включая значения map-секций и пароль из `postgres.DSN`. Сам DSN регистрируется целиком, а пароль из него — отдельно, через `url.Parse`. В `runlog` текст ошибки проходит через `Redact` до обрезки и записи в БД и до отправки уведомления. В `retry.Do` ошибка в warn-логе повтора тоже пишется через `Redact(err.Error())`: ошибка telego — это `*url.Error` с URL запроса, а в URL токен бота (найдено ревью безопасности коммита `sync-delivery-retry`, 2026-10-24). Глобальное состояние выбрано потому, что секреты процесса одни, а прокидывать redactor через fx в runlog ради одной функции — лишний код. (агент)

### D4. `Validate` (Q22)
`Config.Validate()` собирает ошибки через `errors.Join` в формате `<yaml.path>: не задано`. Правила (ключевое поле → обязательные):
- всегда: `telegram.telegram_token`; `database`: задан `dsn` или `host`+`user`+`name` (точные поля по `postgres.Config`);
- `smtp.host` → `smtp.email`, `smtp.password`, `smtp.port`;
- `smb.address` (или ключевое поле адреса в `smb.Config`) → `smb.user`, `smb.password`, `smb.share`;
- `max.enabled` → `max.max_token`;
- `jira.base_url` (ключевое поле адреса) → `jira.auth_token`;
- `appmetrica.o_auth_token` задан → обязательные поля appmetrica, если они есть;
- map-секции (`databases`, `prometheus`, `grafana`, `zabbix`) уже проверяются в конструкторах коллекторов и роняют старт. Их проверка переносится вызовом тех же функций проверки, если они вынесены, иначе остаётся в конструкторах.

Уточнено при реализации (2026-10-24): ключевое поле SMB — флаг `smb.active` (у `smb.address` есть `env-default`, по адресу нельзя понять, включена ли секция), обязательны `address`, `user`, `password`, `share`; флаг Max — `max.max_enabled`; БД — `database.host`, `user`, `name` (`dsn` имеет `yaml:"-"`, пароль не обязателен: допустима trust-аутентификация); у AppMetrica кроме токена обязательных полей нет — правила нет; map-секции остаются в конструкторах коллекторов (отдельных функций проверки нет). Точные имена полей берутся из структур: builder сверяет их с yaml-тегами и проставляет пути по тегам. Уже существующие `Log.Validate()` и `HTTP.Validate()` остаются в составе. (агент)

### D5. live-server
`cmd/live-server/main.go` читает JSON-конфиг с теми же типами секций, поэтому меняется только там, где строки секретов используются напрямую.

## Risks / Trade-offs

- [Новый секрет объявят как `string`] → комментарий над `Config`: «пароли и токены — только `secret.String`», плюс тест, который рефлексией ищет в `Config` строковые поля с `token|password|dsn|secret` в имени и требует тип `secret.String`.
- [`Redact` пропускает секрет, который подставили не из конфига] → вне объёма.
- [Строгая проверка ломает старт на текущих рабочих конфигах] → перед деплоем прогнать `sbot config validate` на рабочем конфиге (Migration Plan).

## Migration Plan

1. До выкладки выполнить `sbot config validate -config <рабочий конфиг>` новым бинарником и дополнить конфиг, если есть ошибки.
2. Выкладка обычным релизом. Откат: предыдущий бинарник, формат конфига не изменился.
