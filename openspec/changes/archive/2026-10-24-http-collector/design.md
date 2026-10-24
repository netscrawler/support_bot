# Design

## Context

Мотивация описана в proposal.md, требования — в `specs/http-collector/spec.md` и `specs/db-secrets/spec.md`. Решения пользователя получены в обсуждении 2026-10-24 (помечены «пользователь»); решения агента помечены «агент».

Текущее устройство, на которое опирается дизайн:
- `collector.Collector.collect` (`internal/collector/collector.go`) берёт `Fetchers[card.Type]` и вызывает `Fetch(ctx, card.CardUUID, card.Query, card.Params)`. Карточку целиком коллекторы не видят.
- `models.Card` (`internal/models/report.go`) — `CardUUID, Title, RawParams, Params, Type, Query`. В БД — таблица `queries(card_uuid, title, q_type, params, query)`.
- `ReportStore.getOrCreateQuery` переиспользует существующую строку `queries` по `(card_uuid, title, query)` (`FindQueryIDByUUIDAndTitle`).
- `secret.Register` / `secret.Redact` (`internal/pkg/secret`) — глобальный список значений, который вычищается из текстов ошибок журнала и уведомлений.
- Lua: `stdlib.CollectPlugin` с map `queries` (`collector.<name>.collect(target, query, params)`), регистрация в `internal/app/module_report.go`.
- `retry.NewRoundTripper` — ретраи транспортных ошибок для HTTP-коллекторов.

## Goals / Non-Goals

**Goals:**
- Один коллектор `http` для GET-API с JSON / NDJSON / CSV.
- Источник описывается в конфиге или в БД, оба пути работают (пользователь).
- Общие `options` в `queries`, которые позже могут читать другие коллекторы (пользователь).
- Секреты в БД без открытых значений (пользователь).

**Non-Goals:**
- POST с телом, XML, пагинация, OAuth2, ретраи по 5xx.
- Ротация `SECRETS_KEY`.
- Ограничение хостов (allowlist) — пользователь явно выбрал «без ограничений».
- Чтение `options` другими коллекторами в этом изменении.
- Учёт `params` при поиске существующей строки `queries` (текущее поведение, см. Risks).

## Decisions

### D1. Схема (пользователь: расширить `queries`, отдельная таблица секретов)
Миграция `migrations/init.sql/011_query_options_secrets.sql`:
```sql
create table secrets (
  name       text primary key,
  value      text not null,              -- base64(nonce || ciphertext)
  updated_at timestamptz not null default now()
);
alter table queries
  add column options jsonb not null default '{}',
  add column secret  text references secrets(name) on delete restrict;
```
Добавить в `schema` в `sqlc.yaml`. На существующей БД накатить вручную.

Агент: ссылка по имени (`secrets.name` — PK), а не по числовому id. В DSL и `ctl export` видно читаемое имя, `on delete restrict` всё так же не даёт удалить используемый секрет. Альтернатива `secret_id int` требовала бы перевода имени в id при `apply` и обратно при `export`.

### D2. Карточка и DSL
`models.Card` + `Options json.RawMessage \`json:"options,omitempty"\`` и `Secret string \`json:"secret,omitempty"\``. `ListCardsByReportID` и `CreateQuery` читают и пишут обе колонки. `FindQueryIDByUUIDAndTitle` дополнительно сравнивает `options` (jsonb `=`) и `secret is not distinct from`, иначе смена опций в `ctl apply` молча переиспользует старую строку. Пустые `Options` пишутся как `{}`.

`ctl apply` с несуществующим секретом падает на FK; store переводит ошибку FK `queries_secret_fkey` в `секрет <name> не найден` (имя берётся из карточки).

### D3. Передача карточки в коллектор (агент)
Новый интерфейс в `internal/collector/collector.go`:
```go
// CardFetcher получает карточку целиком (options, secret).
type CardFetcher interface {
    FetchCard(ctx context.Context, card models.Card) ([]map[string]any, error)
}
```
`collect`: если `f.(CardFetcher)` — вызвать `FetchCard(ctx, crd)` (params уже в `crd.Params`), иначе — `Fetch` как раньше. Коллектор `http` реализует оба метода: `Fetch` (для Lua и как `QueryFetcher` в `Fetchers`) и `FetchCard`.
Альтернатива — поменять сигнатуру `QueryFetcher` — трогает 7 коллекторов, Lua-плагин и моки ради одного потребителя.

### D4. Коллектор `internal/collector/httpcoll`
- `Config` (map-секция `http_sources` в конфиге (`http` уже занята API-сервером — решение агента), ключ — имя источника): `BaseURL string`, `Auth string` (`bearer|basic|header`), `Token secret.String`, `User string`, `Password secret.String`, `AuthHeader string`, `Headers map[string]string`, `Timeout time.Duration` (по умолчанию 30s). Токен и пароль маскируются в `config.LogValue` и регистрируются в `secret.Register`, как у zabbix.
- `New(cfgs map[string]Config, secrets SecretGetter, log) (*Collector, error)` — проверяет `base_url` (абсолютный http/https) и `auth`. `SecretGetter` — интерфейс `Get(ctx, name) (string, error)` объявляется в пакете `http` (потребитель), реализуется store.
- Разрешение источника, порядок слияния (конфиг → options карточки → секрет карточки) и авторизация — как в spec. `options` разбираются в структуру с `DisallowUnknownFields`.
- Один `http.Client` на коллектор с `retry.NewRoundTripper(http.DefaultTransport)`, таймаут — через `context.WithTimeout` на запрос (разный у источников).
- Тело читается через `io.LimitReader(body, 50<<20+1)`; если прочитано больше 50 МБ — ошибка.
- Не 2xx: `fmt.Errorf("http %s: status %d: %s", host, code, первые 512 байт)`. В текст ошибки не попадают заголовки запроса и query string (в query может быть ключ API).
- Значение секрета из БД после расшифровки регистрируется в `secret.Register` — так оно вычищается из журнала и уведомлений тем же механизмом, что и конфиг-секреты (spec db-secrets «Секреты из БД не попадают в ошибки»). Коллектор дополнительно делает `strings.ReplaceAll(err, value, "***")` в собственной ошибке, чтобы не зависеть от порядка.

### D5. Разбор ответа (`decode.go`)
- Выбор формата: `options.format`, иначе `mime.ParseMediaType(Content-Type)` — как в appmetrica.
- json: `json.Decoder` с `UseNumber()` нет — числа как `float64`, как во всех JSON-коллекторах. `root` — обход `map[string]any` по ключам через точку.
- Агент: вложенные объекты разворачиваются `funcs.Flatten`, как у zabbix — экспортёры xlsx/csv ожидают плоские строки. Это отклонение от первоначального плана («без расплющивания»), сделано для единообразия с zabbix.
- ndjson: `bufio.Scanner` с буфером до лимита тела, пустые строки пропускаются.
- csv: свой небольшой парсер в пакете (`encoding/csv`, BOM, `Comma` из `delimiter`, `FieldsPerRecord = 0` — ошибка при несовпадении числа полей). Код appmetrica не трогается (агент: вынос общего парсера — отдельный рефактор).

### D5a. Дополнительные форматы (пользователь: xml, xlsx, text, yaml)
- yaml: `gopkg.in/yaml.v3` (уже прямая зависимость) → `any`, дальше тот же путь, что json (`root`, `row`). Ключи не-строки приводятся `fmt.Sprint`.
- xml: `encoding/xml` токенизатор строит дерево элементов (имя без префикса пространства имён, атрибуты, дети, текст); элемент → `map[string]any` по правилам spec (повторяющиеся дети → `[]any`), затем `funcs.Flatten` — так индексы и точки получаются как у json/zabbix. Без DTD/entity expansion (encoding/xml их не раскрывает).
- xlsx: `github.com/xuri/excelize/v2` (уже прямая зависимость, экспортёр) `OpenReader` → `GetRows(sheet)`; опция `sheet` в options.
- text: `bufio.Scanner` по строкам, `{line: ...}`.
- Решение агента: имена колонок xml-атрибутов `@name`, текста `#text` — распространённая конвенция (xml2json).

### D6. Секреты (пользователь: AES-GCM + ключ из env; привязка к хосту — пользователь, после security review)
- Привязка к хосту: колонка `secrets.host text not null` (в миграции 011). AAD = `name + "\x00" + host`, поэтому подмена `host` или `name` в таблице ломает расшифровку. `SecretStore.Get` возвращает `(value, host)`, `SecretGetter.Get` в `httpcoll` — тоже. Коллектор перед авторизацией сравнивает хост запроса: `strings.EqualFold(host, u.Host)`, а если в хосте секрета нет порта — `strings.EqualFold(host, u.Hostname())`; иначе ошибка без запроса. Редиректы на другой хост уже запрещены (`sameHostRedirect`). Без этого любой с правом записи в БД (без `SECRETS_KEY`) мог бы получить токен, направив карточку на свой сервер.
- `internal/pkg/secret/cipher.go`: `type Cipher struct{ aead cipher.AEAD }`, `NewCipher(b64key string) (*Cipher, error)` (ровно 32 байта), `Encrypt(plain) (string, error)`, `Decrypt(b64) (string, error)`. Nonce — `crypto/rand`, хранится перед шифротекстом. AAD — имя секрета, чтобы значение нельзя было переставить в чужую строку.
- Конфиг: `Secrets struct{ Key secret.String \`env:"SECRETS_KEY" yaml:"key"\` }`. Пустой ключ — `Cipher` = nil; неверная длина — ошибка `config.Validate`.
- `internal/store/secret_store.go` + `internal/db/queries/secret.sql` (`UpsertSecret`, `GetSecret`, `ListSecretNames`, `DeleteSecret`). `SecretStore.Get` расшифровывает; при nil `Cipher` — ошибка `SECRETS_KEY не задан`. `Delete` переводит ошибку FK в «секрет используется запросами».
- CLI `ctl secret set|list|delete` в `internal/cli/ctl.go` (рядом с `script`), значение `set` — `io.ReadAll(os.Stdin)` с обрезкой завершающего `\n`.

### D7. Lua (агент: без секретов)
`collector.http.collect(target, query, params, options)`: `options` — необязательная Lua-таблица, конвертируется `luapkg.LuaTableToGo` → JSON → `Card.Options`, вызывается `FetchCard`. Ключ `secret` в options отклоняется до запроса. Регистрировать `http` в Lua-`queries` всегда (URL-источники не требуют конфига). Изменение в `internal/processor/lua/stdlib/` — после реализации `/security-review`.
Если текущий `CollectPlugin` не позволяет четвёртый аргумент без изменения общего пути — добавить отдельную ветку только для `http` (минимально), не трогая остальные коллекторы.

### D8. Валидация (`internal/service/report_validator.go`)
Для `type = http`: `card_uuid` не пуст; если в нём есть `://` — парсится `url.Parse`, схема http/https, есть host; `options` — объект, ключи из списка D4, `format` ∈ {json, ndjson, csv}, `auth` ∈ {bearer, basic, header}. `auth_header` не обязателен — может прийти из конфига источника (решение агента при реализации); без него ошибка при сборе. Имена конфиг-источников валидатор не знает — неизвестное имя ловится при сборе (spec «Не URL и не имя из конфига»).

### D8a. Wiring
`internal/app/module_collectors.go`: провайдер `newHTTPCollector` (ошибка `New` — старт падает, как у zabbix), `models.CollectTypeHTTP: httpColl` в `Fetchers`. `cmd/live-server/main.go`: секция `http_sources` и коллектор, `SecretGetter` = nil (секреты из БД в live-server не поддерживаются — ошибка при карточке с `secret`). `config/config.example.yaml`, `config/example.env` — примеры.

### D9. Тесты
- `http`: `httptest.Server` — конфиг-источник и URL, перекрытие заголовков, bearer/basic/header из конфига и из секрета (фейковый `SecretGetter`), секрет не найден, json+root/скаляры/нет root, ndjson, csv с BOM и `;`, несовпадение полей, не 2xx с обрезкой тела, токен в теле ошибки → `***`, лимит 50 МБ (сервер пишет 50 МБ + 1), неизвестный Content-Type, неизвестная опция.
- `collector`: `collect` вызывает `FetchCard`, если реализован.
- `secret`: roundtrip, чужой ключ, чужое имя/хост (AAD), ключ не 32 байта; `httpcoll`: секрет на чужой хост — ошибка без запроса.
- `store`: `secret_store` через `pgxmock` (upsert, get + decrypt, delete с FK-ошибкой); `report_store` — options/secret читаются и пишутся, поиск строки учитывает options.
- `validator`, `config` (маскирование `http_sources.*.token`), Lua (`options`, отказ на `secret`).

## Risks / Trade-offs

- [Нет allowlist хостов — URL из БД может указать на внутренний сервис или метаданные облака] → осознанный выбор пользователя; в БД пишут только админы через `ctl`. Задокументировать в `config.example.yaml`.
- [Потеря `SECRETS_KEY` = потеря всех секретов] → в `config/example.env` комментарий «хранить ключ вне сервера, при потере секреты заводятся заново `ctl secret set`».
- [`options.headers` хранится открыто — туда могут вписать токен] → в example и в README коллектора: токены только через `secret`.
- [Карточка логируется целиком (`slog.Any("card", crd)` в `collect`) — попадут `options` и имя секрета, но не значение] → приемлемо; значение секрета в карточке не хранится.
- [`getOrCreateQuery` не учитывает `params` при поиске — существующее поведение, смена params в `apply` может не примениться] → вне scope; options/secret учитываются (D2).
- [`secret.Register` — глобальный список растёт с каждым новым секретом] → число секретов мало, дубликаты отбрасываются.
- [Ответ 50 МБ целиком в памяти] → лимит; при необходимости — потоковый разбор позже.

## Migration Plan

1. На существующей БД выполнить `011_query_options_secrets.sql` вручную (колонки с дефолтами — старые запросы не меняются).
2. Сгенерировать ключ `openssl rand -base64 32`, положить в `SECRETS_KEY` (env/секрет деплоя).
3. Деплой бота. Без ключа и секции `http_sources` бот работает как раньше.
4. Откат: старый бинарник игнорирует новые колонки; таблицу и колонки можно оставить.
