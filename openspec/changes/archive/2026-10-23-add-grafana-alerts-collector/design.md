# Design

## Context

Изменение `add-prometheus-collector` уже добавило несколько вещей. Секцию конфига `grafana: map[name]prometheus.GrafanaConfig{URL, Token, Timeout}` с маскированием токена в `LogValue`. Общий интерфейс `QueryFetcher.Fetch(ctx, target, query, params)` в `internal/collector`. В Lua-плагине `queries map[string]QueryCollector`: каждый элемент регистрируется как `collector.<name>.collect(target, query, params)`. Новый коллектор встраивается через эти же механизмы.

Мотивация описана в proposal.md, требования — в `specs/grafana-alerts-collector/spec.md`. Решения взяты из grilling 2026-10-23: Q7, Q8, Q11, Q13–Q15 в плане `/home/netscrawler/.claude/plans/gleaming-brewing-bengio.md` и Q19–Q21, заданные перед этим изменением.

## Goals / Non-Goals

**Goals:**
- Три режима поверх двух endpoint'ов Grafana, без новых зависимостей.
- Встроить коллектор в существующие switch, `QueryFetcher` и Lua `queries`.

**Non-Goals:**
- Alertmanager API (`/api/alertmanager/grafana/api/v2/alerts`). Он не отдаёт pending и value (Q19), поэтому информацию о silence не показываем.
- История из Loki или Prometheus-backend. Для Prometheus-backend хватит запроса `prom` по `GRAFANA_ALERTS`.
- Разбор labels из `text` аннотаций (Q20): формат нестабилен.
- Серверные фильтры и пагинация rules API (Q21): их набор зависит от версии Grafana.

## Decisions

### D1. Пакет и конфиг
Новый пакет `internal/collector/grafana`: `New(grafanas map[string]prometheus.GrafanaConfig, log)` и `Fetch(ctx, target, mode, params)`. Тип конфига остаётся в пакете `prometheus`, переносить его не стоит: перенос затронул бы `config.go`, live-server и тесты прошлого изменения ради одного имени. Новых секций конфига нет, у `Grafana` в `internal/config/config.go` меняется только комментарий: секция теперь нужна и для алертов. URL обрезается справа от `/`. Авторизация — `Authorization: Bearer <token>`. Timeout вычисляется как `cmp.Or(g.Timeout, 30s)`. HTTP-клиент: `retry.NewRoundTripper(http.DefaultTransport)` и `context.WithTimeout` на каждый запрос, как в `prometheus/collector.go`.
- Альтернатива — вынести `GrafanaConfig` в `grafana.Config`. Отложена, пока нет третьего потребителя.

### D2. Хранение и Lua (Q13, Q15)
- `q_type = 'grafana_alerts'` (`models.CollectTypeGrafanaAlerts`), `card_uuid` = имя Grafana, режим хранится в `queries.query`. Миграции нет.
- Ветка в switch: `c.alerts.Fetch(ctx, target, crd.Query, params)`, новый аргумент `NewCollector`.
- В Lua `queries["grafana_alerts"]` добавляется только при `len(cfg.Grafana) > 0`, через интерфейсную переменную, чтобы обойти ловушку typed nil (D8 прошлого изменения).

### D3. Режимы active и rules: один запрос (Q19)
Оба режима делают `GET <url>/api/prometheus/grafana/api/v1/rules`. Ответ: `{status, data: {groups: [{name, file, rules: [{name, state, health, lastEvaluation, labels, alerts: [{labels, annotations, state, activeAt, value}]}]}]}}`. Поле `file` содержит путь папки, из него берётся колонка `folder`.
- `rules`: одна строка на правило — `{folder, group, rule, state, health, last_evaluation, active_instances: len(alerts), <labels правила>}`.
- `active`: одна строка на экземпляр из `alerts[]` — `{alertname: rule.name, state, active_at, value, folder, group, <labels экземпляра>, annotation_<k>}`. `value` — строка оценки Grafana как есть (`[ var='B' labels={...} value=1 ]`), не число.
- Проверка на Grafana 11.2.2 показала, что `alerts[]` содержит и экземпляры в `Normal` (`Normal (MissingSeries)` и т.п.). Экземпляры со state, начинающимся на `Normal`, отбрасываются до построения строк. Поэтому `active` показывает только активные экземпляры, а `active_instances` считает только их.
- Служебные колонки перезаписывают labels с теми же именами, как у prom. Правила без `alerts` в `active` строк не дают. Время приводится к RFC3339 UTC. Если время нулевое (`0001-01-01…`, правило ещё не оценивалось), колонка равна `nil`.
- Ответ `status != "success"` даёт ошибку `grafana <name>: <errorType>: <error>`.

### D4. Режим history (Q11, Q20)
`GET <url>/api/annotations?type=alert&from=<ms>&to=<ms>&limit=<n>`.
- `from` и `to` необязательны. Число передаётся как есть. Иначе значение разбирается через `time.Parse(time.RFC3339, …)` и переводится в `UnixMilli`. Ошибка разбора содержит имя параметра.
- `limit` по умолчанию 10000: дефолт Grafana 100 молча отрезал бы данные. Значение должно быть положительным целым, иначе ошибка.
- Строка: `{time, time_end, alertname, prev_state: prevState, new_state: newState, text}`. Проверка на Grafana 11.2.2 показала, что unified alerting оставляет `alertName` пустым, а `text` имеет вид `<имя правила> {<labels>} - <значения>`. Поэтому `alertname = alertName`, а если оно пустое, берётся `text` до первого ` {`. Пользователь согласовал это 2026-10-23. Labels из `text` по-прежнему не разбираем. `time` и `timeEnd` приходят в epoch ms и переводятся в RFC3339. Если `timeEnd` равно 0, колонка равна `nil`.

### D5. Фильтры и ключи params (Q21)
Допустимые ключи по режимам:
- `active`: `folder`, `group`, `state`, `alertname`.
- `rules`: `folder`, `group`, `state`, `alertname`. Здесь `alertname` сравнивается с именем правила.
- `history`: `from`, `to`, `limit`, `alertname`.

Перед запросом проверяется, что все ключи допустимы. Недопустимый ключ даёт ошибку с его именем, неизвестный режим — ошибку с перечнем режимов, и запрос в обоих случаях не отправляется. Фильтры сравниваются точно, без регистра и префиксов, с полем строки до переименования labels. Значения state в Grafana разные: у правил `inactive`/`pending`/`firing`, у экземпляров `Alerting`/`Pending`/`Normal`/`Alerting (NoData)` и т.п. Поэтому значения берутся из самих данных (см. Risks).

### D6. Ошибки HTTP
Тело читается целиком. Если статус вне 2xx, возвращается `grafana <name>: HTTP <code>: <первые 200 байт тела>`. Токен находится только в заголовке запроса, в тело и URL он не попадает. Если тело 2xx не разбирается, возвращается ошибка со статусом. Для rules API тело сначала разбирается как Prometheus-ответ: тогда при 4xx/5xx виден текст `error` (так же, как у prom).

### D7. Тесты
- `internal/collector/grafana`, `httptest.Server`:
  - путь и заголовок Bearer;
  - `rules`: строки и `active_instances`, в том числе 0;
  - `active`: firing и pending, annotations с префиксом, `value` как строка;
  - фильтры;
  - недопустимый ключ и неизвестный режим без запроса к серверу;
  - `history`: RFC3339 и ms переводятся в query, дефолт `limit=10000`, плохой `from`, `timeEnd = 0`;
  - 403 с JSON-телом;
  - `status: error` у rules;
  - неизвестная Grafana.
- `collector_test.go`: ветка `grafana_alerts` вызывает мок `QueryFetcher` с `(CardUUID, Query, Params)`.
- `stdlib/collector_test.go`: менять не нужно, механизм `queries` уже покрыт. Проверка «без секции нет модуля» живёт в провайдере `module_report.go`, тестов у пакета `internal/app` нет, она проверяется ручным прогоном.

## Risks / Trade-offs

- [Lua и запросы видят все алерты, доступные токену] → нужен service account с ролью Viewer. Rules API сам отфильтрует папки без прав. Перед archive обязателен `/security-review`.
- [Разные словари state у правил и экземпляров, суффиксы вида `Alerting (NoData)`] → фильтр работает точно по значению из данных. В `config.example.yaml` нужен комментарий с примерами значений.
- [Rules API в новых версиях может отдавать `alerts[]` не полностью (лимиты экземпляров) или поддерживать пагинацию] → без параметров Grafana отдаёт все группы. Если обнаружится усечение, добавить нужный query-параметр.
- [В `history` нет labels экземпляра, они только внутри `text`] → принимаем (Q20).
- [Лимит 10000 тоже может отрезать данные за большой период] → его можно переопределить через params. Бот не сообщает об упоре в лимит.

## Migration Plan

1. Секция `grafana` в рабочем конфиге уже нужна для prom-через-Grafana. Если её нет, добавить. Токену Grafana нужны права на чтение правил и аннотаций (Viewer).
2. Завести запросы `q_type = 'grafana_alerts'`, миграции БД нет.
3. Откат: старый бинарник ответит «unknown fetcher type» на этот тип, остальные запросы не затрагиваются.
