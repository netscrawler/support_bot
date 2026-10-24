# Design

## Context

Как собираются данные сейчас. `collector.Collector` выбирает fetcher по `models.Card.Type`. У pg свой интерфейс `SQLFetcher.Fetch(ctx, target, query, params)`: `target` берётся из `card_uuid`, `query` из `queries.query`, а `params` уже разрешены через `Card.ResolveParams`. В Lua коллекторы попадают через `luastd.CollectPlugin`: `directs` для `(target, params)` и отдельное поле `pg`. HTTP-коллекторы (jira) используют `retry.NewRoundTripper(http.DefaultTransport)`. Секреты маскируются в `config.Config.LogValue`, map копируется, как у `Databases`.

Мотивация описана в proposal.md, требования — в `specs/prometheus-collector/spec.md`. Решения ниже договорены с пользователем на grilling-сессии 2026-10-23 (Q1–Q18).

## Goals / Non-Goals

**Goals:**
- Один клиент и один парсер для Prometheus API и для datasource proxy Grafana.
- Встроить новый коллектор в существующий switch и Lua-плагин так же, как pg.

**Non-Goals:**
- Алерты Grafana. Это отдельное изменение `add-grafana-alerts-collector`, оно будет переиспользовать секцию `grafana`.
- Чтение запросов из панелей дашбордов Grafana (вариант C), Loki и LogQL.
- `/api/ds/query` Grafana (data frames).
- Собственный CA и `insecure_skip_verify`: используются только системные CA.
- Свой лимит точек. Хватает лимита Prometheus (11000 точек на серию) и timeout.
- Native histograms. Сэмплы с полем `histogram` вместо `value` дают `value: null` (см. Risks).

## Decisions

### D1. Два транспорта, один клиент (Q1, Q5)
Через datasource proxy Grafana (`<grafana.url>/api/datasources/proxy/uid/<uid>/api/v1/...`) приходит тот же JSON, что и от самого Prometheus. Поэтому при старте каждая запись `prometheus` превращается в пару `baseURL` + функцию авторизации:
- напрямую: `baseURL = url`, заголовок `Authorization: Bearer <bearer_token>` или `SetBasicAuth`, либо без авторизации;
- через Grafana: `baseURL = <grafana.url>/api/datasources/proxy/uid/<datasource_uid>`, заголовок `Authorization: Bearer <grafana.token>`.

Дальше запросы идут одинаково: `GET <baseURL>/api/v1/query` или `/api/v1/query_range`. Используется GET, а не POST-форма. Это допущение, его никто не проверял: GET принимают и Prometheus, и любой proxy, а POST через proxy Grafana может зависеть от версии и настроек. Минус GET — лимит длины URL (см. Risks).
- Альтернатива: `/api/ds/query`. Отклонена, для неё нужен второй парсер data frames.

### D2. Конфиг (Q4, Q6, Q17)
```yaml
grafana:
  ops:
    url: https://grafana.example
    token: glsa_...          # service account token, роль Viewer
    timeout: 30s             # default 30s
prometheus:
  main:
    url: https://prom.example
    bearer_token: ...        # опционально
    basic_auth: {user: u, password: p}   # опционально, взаимоисключимо с bearer_token
    timeout: 30s             # default 30s
  main_via_grafana:
    grafana: ops
    datasource_uid: P1809F7CD0C75ACF3
```
Типы: `map[string]prometheus.Config` и `map[string]prometheus.GrafanaConfig`. Оба живут в пакете `internal/collector/prometheus`. В следующем изменении `GrafanaConfig` можно будет вынести, если коллектору алертов понадобится свой пакет. Дефолты для значений map ставятся в коде, как у pg, потому что `env-default` для map не работает. Валидация выполняется в `prometheus.New(cfgs, grafanas)`. Возвращаемая ошибка роняет старт через fx. Проверяется: задан ровно один из вариантов (`url` или `grafana`+`datasource_uid`), grafana-ссылка существует, нет одновременно `bearer_token` и `basic_auth`. В `LogValue` обе map копируются, а `token`, `bearer_token` и `basic_auth.password` заменяются на `***`. Timeout у proxy-записи берётся из её собственного `timeout`, а если он не задан — из timeout Grafana.

### D3. Хранение (Q10)
Здесь всё как у pg: `q_type = 'prom'`, `card_uuid` = имя из `prometheus`, PromQL лежит в `queries.query`. Миграций нет. Дедупликация `(card_uuid, title, query)` уже работает для всех типов.

### D4. Instant или range (Q3)
Если в params есть все три `start`, `end`, `step`, запрос идёт в `query_range`. Если нет ни одного — в `query`, с `time`, если он задан. Если задана только часть, это ошибка с перечнем недостающих параметров. Это моё допущение: молча сделать instant при опечатке в `step` хуже, чем упасть. Значения передаются как есть. Prometheus сам принимает RFC3339 и unix-время, а `step` — как duration или секунды. Даты считает существующий `ResolveParams`.

### D5. Подстановка `${name}` (Q10, Q12)
До запроса все вхождения `\$\{(\w+)\}` заменяются значением `params[name]`, без экранирования. Подставляются и API-параметры (`${step}` в `[...]`). Если какого-то плейсхолдера нет в params, возвращается ошибка с его именем, запрос не отправляется. В API уходят только `start`, `end`, `step`, `time`, остальные params нужны только для подстановки.
- Голый `$name` не поддерживаем, потому что он неоднозначен (`$svc_total`).
- Экранирования нет: Prometheus API только читает, а запросы и params пишет админ. Если params когда-нибудь начнёт задавать пользователь бота, понадобится экранирование `"` и `\` внутри строковых литералов (см. Risks).

### D6. Форма строк (Q9)
- `vector` → одна строка на серию: `labels` + `value`.
- `matrix` → одна строка на точку: `labels` + `timestamp` + `value`.
- `__name__` переименовывается в `metric`. Если у серии есть label `value` или `timestamp`, служебное поле её перезаписывает.
- `value` — `strconv.ParseFloat`. NaN и ±Inf превращаются в `nil`, потому что дальше значения идут через JSON и в Lua, и NaN там ломает `json.Marshal`.
- `timestamp` — unix float, переводится в `time.Time` и форматируется как RFC3339 в UTC. Строка, а не `time.Time`, нужна для единообразия с pg (D7 pg-коллектора).
- У строк разных серий может быть разный набор ключей.
- `scalar` и `string` дают ошибку о неподдерживаемом типе результата.

### D7. Ошибки и ретраи (Q18)
Тело ответа декодируется в `{status, data{resultType, result}, errorType, error, warnings}` при любом HTTP-статусе, потому что Prometheus отдаёт JSON-ошибку с 400/422/503. Если `status != "success"`, возвращается ошибка `prometheus <name>: <errorType>: <error>`. Если тело не JSON (так бывает при 401/403/502 от Grafana или прокси), возвращается ошибка со статусом. `warnings` пишутся в лог как warn с именем источника. HTTP-клиент: `&http.Client{Transport: retry.NewRoundTripper(http.DefaultTransport), Timeout: timeout}`, как у jira.

### D8. Интерфейсы: переиспользуем pg-сигнатуру
У prom та же сигнатура `Fetch(ctx, target, query, params)`, что у pg. Поэтому `SQLFetcher` переименовывается в `QueryFetcher` (`internal/collector`), а `SQLCollector` — в `QueryCollector` (`internal/processor/lua/stdlib`). Комментарий: «выполняет запрос `query` в именованном источнике `target`». Моки перегенерируются через `go tool mockery`. Изменения в `collector.Collector`:
- новое поле `prom QueryFetcher`;
- новый аргумент `NewCollector`;
- ветка `case models.CollectTypePrometheus` (`"prom"`), которая вызывает `c.prom.Fetch(ctx, crd.CardUUID, crd.Query, crd.Params)`.

В `CollectPlugin` поле `pg` меняется на `queries map[string]QueryCollector`, регистрирующее `collector.<name>.collect(target, query, params)` для каждого элемента. pg и prom подключаются как `{"pg": ..., "prom": ...}`, и следующий коллектор алертов (mode в позиции query) встанет туда же без нового кода. Записи с nil-значением не регистрируются: так сохраняется прежнее правило «нет pg — нет `collector.pg`». Нужна осторожность с typed nil: `*prometheus.Collector`, положенный в map, никогда не равен nil-интерфейсу. Поэтому в `newLuaStdCollector` prom кладётся в map только при непустом `cfg.Prometheus`, через переменную интерфейсного типа, как сейчас сделано для pg (`module_report.go:144`).
- Альтернатива: отдельное поле `prom` рядом с `pg`. Отклонена, это вторая копия `luaPGCollect`.

### D9. Подключение
- `internal/app/module_collectors.go`: провайдер `newPrometheusCollector(cfg, log)` вызывает `prometheus.New(cfg.Prometheus, cfg.Grafana, log)` и передаёт результат в `newCollector`.
- `internal/app/module_report.go`: `prom` добавляется в `queries` Lua-плагина.
- `cmd/live-server/main.go`: в его json-конфиг добавляются поля `prometheus` и `grafana`, и та же передача в `NewCollector`.

Отдельной проверки `q_type` при создании отчёта нет: `report_validator.go:40` проверяет только `mb`, а store сохраняет тип как есть. Поэтому для `prom` больше ничего менять не нужно.

Если секция `prometheus` пустая, коллектор всё равно создаётся. Запросы `prom` тогда падают с «unknown prometheus source», так же как pg без баз.

### D10. Тесты
- `internal/collector/prometheus`: `httptest.Server` проверяет путь и query-параметры (instant, range, `time`), подстановку и ошибку неизвестного плейсхолдера, неполный range, vector/matrix → строки, `__name__` → `metric`, NaN → nil, `status:error` → текст ошибки, scalar → ошибка, не-JSON 502 → ошибка со статусом, warnings не мешают данным, proxy-URL и Bearer Grafana, basic auth. Валидация конфига табличным тестом.
- `internal/collector/collector_test.go`: ветка `prom` вызывает `QueryFetcher` с `(CardUUID, Query, Params)`.
- `internal/processor/lua/stdlib/collector_test.go`: `collector.prom.collect` доходит до мока с аргументами как есть, ошибка возвращается вторым значением. Существующие pg-тесты проходят на новом `queries`.
- `internal/config/config_test.go`: `LogValue` маскирует три вида секретов и не портит исходный конфиг.

## Risks / Trade-offs

- [Lua и запросы видят все метрики всех источников конфига] → токены выдавать с минимальными правами (service account Grafana с ролью Viewer, при необходимости только на нужные datasource). Это пишется в комментарий `config.example.yaml`. Перед archive обязателен `/security-review`.
- [Подстановка без экранирования] → допустимо, пока params задаёт админ, и API только читает. Если появится пользовательский ввод в params, нужно экранирование (D5).
- [GET и длинный PromQL упираются в лимит URL (~8 KB у прокси)] → для отчётных запросов этого хватает. Если нет, можно перейти на POST-форму для прямого доступа.
- [Range с маленьким `step` даёт много строк] → лимит Prometheus 11000 точек на серию и timeout. Свой лимит не делаем, как у pg.
- [Native histograms дают `value: null` без ошибки] → принимаем, такие метрики надо запрашивать через `histogram_quantile` и подобные функции.
- [Label `value` или `timestamp` перезаписывается служебным полем] → редкость, принимаем.
- [PromQL, скопированный из панели Grafana, содержит её переменные: `${__range}` упадёт как неизвестный плейсхолдер, а голый `$__rate_interval` уйдёт в Prometheus и вернётся ошибкой парсинга] → напомнить об этом в комментарии `config.example.yaml`. Нужное значение передаётся явным param, например `${step}`.
- [Таймаут proxy-записи без своего `timeout` берётся у Grafana] → дефолт выбран мной, пользователь его не согласовывал.

## Migration Plan

1. Добавить секции `grafana` и `prometheus` в рабочий конфиг. Без них бот работает как раньше, запросы `prom` падают с «unknown prometheus source».
2. Завести запросы `q_type = 'prom'`. Миграции БД нет.
3. Откат: старый бинарник не знает тип `prom` и возвращает «unknown fetcher type». Остальные запросы отчёта не затрагиваются.
