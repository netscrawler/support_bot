# grafana-alerts-collector Specification

## Purpose
Позволяет собирать в отчёты данные об алертах Grafana-managed alerting: активные экземпляры, состояние правил и историю срабатываний из Grafana, описанной в конфиге бота.

## Requirements

### Requirement: Запрос типа grafana_alerts
Запрос отчёта с типом `grafana_alerts` SHALL выполняться в Grafana, имя которой указано в `card_uuid`, в режиме из поля `query`. Результат SHALL попадать в датасет отчёта под заголовком запроса как список строк «колонка → значение».

#### Scenario: Неизвестная Grafana
- **WHEN** `card_uuid` указывает на имя, которого нет в секции `grafana`
- **THEN** сбор этого запроса завершается ошибкой с этим именем, а остальные запросы отчёта собираются как обычно

#### Scenario: Неизвестный режим
- **WHEN** `query = "silences"`
- **THEN** сбор завершается ошибкой, в которой перечислены допустимые режимы `active`, `rules`, `history`

### Requirement: Режим active
В режиме `active` коллектор SHALL возвращать по строке на каждый экземпляр алерта Grafana-managed правил с колонками `alertname`, `state`, `active_at` (RFC3339), `value` (строка оценки как есть), `folder`, `group`, labels экземпляра как отдельные колонки и annotations как колонки `annotation_<ключ>`.

#### Scenario: Firing и pending
- **WHEN** у правила `HighErrors` два экземпляра, firing и pending
- **THEN** результат содержит две строки с `alertname = "HighErrors"` и соответствующими `state`

### Requirement: Режим rules
В режиме `rules` коллектор SHALL возвращать по строке на каждое правило с колонками `folder`, `group`, `rule`, `state`, `health`, `last_evaluation` (RFC3339), `active_instances` (число экземпляров) и labels правила как отдельные колонки.

#### Scenario: Правило без экземпляров
- **WHEN** правило в состоянии inactive и экземпляров у него нет
- **THEN** результат содержит строку этого правила с `active_instances = 0`

### Requirement: Режим history
В режиме `history` коллектор SHALL возвращать по строке на каждую смену состояния алерта из аннотаций Grafana с колонками `time`, `time_end` (RFC3339), `alertname`, `prev_state`, `new_state`, `text`. Период задаётся params `from`/`to` в RFC3339 или epoch ms. Параметр `limit` по умолчанию 10000.

#### Scenario: Период в RFC3339
- **WHEN** params = `{"from": "2026-10-01T00:00:00Z", "to": "2026-10-08T00:00:00Z"}`
- **THEN** в Grafana уходит запрос аннотаций типа alert с `from=1790812800000`, `to=1791417600000` и `limit=10000`

#### Scenario: Имя правила из текста аннотации
- **WHEN** у аннотации пустой `alertName`, а `text = "HighErrors {alertname=HighErrors, svc=api} - B=1"`
- **THEN** в строке `alertname = "HighErrors"`

#### Scenario: Неверный формат времени
- **WHEN** `from = "вчера"`
- **THEN** сбор завершается ошибкой с именем параметра

### Requirement: Фильтры
Params SHALL фильтровать строки результата на стороне бота по точному совпадению: `folder`, `group`, `state` (в режимах `active` и `rules`), `alertname` (во всех режимах). Ключ params, недопустимый для режима, MUST давать ошибку с его именем.

#### Scenario: Фильтр по состоянию
- **WHEN** режим `active` и params = `{"state": "Alerting"}`
- **THEN** в результате только экземпляры с `state = "Alerting"`

#### Scenario: Опечатка в ключе
- **WHEN** params = `{"stat": "firing"}`
- **THEN** сбор завершается ошибкой с именем `stat`

### Requirement: Ошибки ответа
Если Grafana вернула HTTP-статус вне 2xx или ответ не удалось разобрать, запрос MUST завершаться ошибкой с HTTP-статусом и началом тела ответа. Токен Grafana MUST не попадать в текст ошибки.

#### Scenario: Нет прав
- **WHEN** Grafana отвечает 403 `{"message": "access denied"}`
- **THEN** сбор завершается ошибкой, содержащей `403` и `access denied`

### Requirement: Вызов из Lua
Lua-скрипты SHALL вызывать коллектор как `collector.grafana_alerts.collect(name, mode, params)` с той же логикой, что у запроса `grafana_alerts`. Функция SHALL быть доступна только при непустой секции `grafana` в конфиге и возвращать `rows, nil` или `nil, err`.

#### Scenario: Нет секции grafana
- **WHEN** в конфиге нет секции `grafana`
- **THEN** в Lua `collector.grafana_alerts` равен nil
