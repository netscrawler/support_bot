# prometheus-collector Specification

## Purpose
Позволяет собирать данные для отчётов PromQL-запросами к Prometheus-совместимым источникам из конфига бота, напрямую или через Grafana.

## Requirements

### Requirement: Запрос типа prom
Запрос отчёта с типом `prom` SHALL выполняться в Prometheus-источнике, имя которого указано в `card_uuid`, с PromQL из поля `query`. Результат SHALL попадать в датасет отчёта под заголовком запроса как список строк «колонка → значение».

#### Scenario: Успешный сбор
- **WHEN** отчёт содержит запрос `prom` с `card_uuid = "main"`, `query = "up"`, и источник `main` описан в конфиге
- **THEN** датасет под заголовком запроса содержит по строке на каждую серию ответа

#### Scenario: Неизвестный источник
- **WHEN** `card_uuid` запроса `prom` указывает на имя, которого нет в секции `prometheus`
- **THEN** сбор этого запроса завершается ошибкой с этим именем, а остальные запросы отчёта собираются как обычно

### Requirement: Instant и range запросы
Если в params есть `start`, `end` и `step`, коллектор SHALL выполнять range-запрос с этими значениями. Иначе он SHALL выполнять instant-запрос, передавая `time`, если он задан. Значения передаются в API Prometheus без преобразования.

#### Scenario: Range-запрос
- **WHEN** params = `{"start": "2026-10-01T00:00:00Z", "end": "2026-10-02T00:00:00Z", "step": "1h"}`
- **THEN** выполняется range-запрос, и в результате по строке на каждую точку каждой серии

#### Scenario: Instant-запрос
- **WHEN** в params нет `start`/`end`/`step`
- **THEN** выполняется instant-запрос, и в результате по строке на каждую серию

#### Scenario: Неполный range
- **WHEN** в params есть только часть из `start`, `end`, `step`
- **THEN** запрос завершается ошибкой, где перечислены недостающие параметры

### Requirement: Подстановка в PromQL
Каждый плейсхолдер `${name}` в PromQL SHALL заменяться значением параметра `name` из params без изменений. Если плейсхолдер ссылается на отсутствующий параметр, запрос MUST завершаться ошибкой. Параметры кроме `start`, `end`, `step`, `time` SHALL использоваться только для подстановки и не передаваться в API.

#### Scenario: Подстановка значения
- **WHEN** `query = "sum(rate(http_requests_total{service=\"${svc}\"}[${step}]))"` и params = `{"svc": "api", "step": "5m", ...}`
- **THEN** в Prometheus уходит запрос с `service="api"` и `[5m]`

#### Scenario: Неизвестный плейсхолдер
- **WHEN** в PromQL есть `${env}`, а параметра `env` в params нет
- **THEN** запрос завершается ошибкой с именем `env` и в Prometheus не отправляется

### Requirement: Форма строк результата
Каждая строка SHALL содержать labels серии как отдельные колонки (`__name__` переименовывается в `metric`) и `value` как число. Значения NaN и ±Inf SHALL становиться null. В range-запросе в строку SHALL добавляться `timestamp` точки в RFC3339 (UTC).

#### Scenario: Строка instant
- **WHEN** серия `{__name__="up", job="api"}` имеет значение `"1"`
- **THEN** строка равна `{"metric": "up", "job": "api", "value": 1}`

#### Scenario: NaN
- **WHEN** значение точки равно `"NaN"`
- **THEN** `value` в строке равно null, а сбор не падает

### Requirement: Ошибки ответа
Если Prometheus вернул `status: error` или ответ не в формате JSON API, запрос MUST завершаться ошибкой с текстом ошибки из ответа. Если результат имеет тип `scalar` или `string`, запрос MUST завершаться ошибкой. Предупреждения (`warnings`) SHALL записываться в лог, данные при этом SHALL возвращаться.

#### Scenario: Ошибка PromQL
- **WHEN** PromQL синтаксически неверен и Prometheus отвечает `status: error`
- **THEN** сбор запроса завершается ошибкой, содержащей текст ошибки Prometheus

#### Scenario: Scalar
- **WHEN** запрос `query = "1+1"` возвращает результат типа `scalar`
- **THEN** сбор запроса завершается ошибкой о неподдерживаемом типе результата

### Requirement: Доступ через Grafana
Источник из секции `prometheus` с полями `grafana` и `datasource_uid` SHALL запрашиваться через datasource proxy Grafana, указанной в секции `grafana`, с её токеном. Результат SHALL быть таким же, как при прямом доступе.

#### Scenario: Запрос через proxy
- **WHEN** источник описан как `{grafana: "ops", datasource_uid: "abc"}`
- **THEN** запрос уходит на `<url ops>/api/datasources/proxy/uid/abc/api/v1/query` с токеном Grafana `ops`

### Requirement: Конфигурация источников
Каждая запись секции `prometheus` MUST задавать ровно один вариант: `url` (необязательно с `bearer_token` или `basic_auth`) или `grafana` + `datasource_uid`, причём grafana-ссылка указывает на существующую запись секции `grafana`. Иначе бот MUST не стартовать и сообщить имя записи. Токены и пароли MUST маскироваться при выводе конфига в лог.

#### Scenario: Оба варианта сразу
- **WHEN** у записи заданы и `url`, и `grafana`
- **THEN** бот не стартует и в ошибке есть имя записи

#### Scenario: Ссылка на несуществующую Grafana
- **WHEN** запись ссылается на `grafana: "ops"`, а в секции `grafana` записи `ops` нет
- **THEN** бот не стартует и в ошибке есть оба имени

#### Scenario: Маскирование секретов
- **WHEN** конфиг выводится в лог
- **THEN** `token`, `bearer_token` и пароль `basic_auth` не видны в открытом виде

### Requirement: Вызов из Lua
Lua-скрипты SHALL вызывать коллектор как `collector.prom.collect(name, promql, params)` с той же логикой, что у запроса `prom`. Вызов SHALL возвращать `rows, nil` при успехе и `nil, err` при ошибке.

#### Scenario: Ошибка из Lua
- **WHEN** скрипт вызывает `collector.prom.collect("unknown", "up")`
- **THEN** первое возвращаемое значение равно nil, второе — строка с ошибкой
