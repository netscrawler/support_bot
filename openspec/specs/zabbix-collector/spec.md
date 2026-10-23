# zabbix-collector Specification

## Purpose
Позволяет собирать в отчёты любые данные, которые Zabbix отдаёт на чтение через свой API (`*.get`), из Zabbix-инсталляций, описанных в конфиге бота.

## Requirements

### Requirement: Запрос типа zabbix
Запрос отчёта с типом `zabbix` SHALL выполняться в Zabbix, имя которого указано в `card_uuid`. Поле `query` SHALL содержать JSON-объект с ключами `method` и `params`, которые передаются в JSON-RPC вызов Zabbix. Результат SHALL попадать в датасет отчёта под заголовком запроса как список строк «колонка → значение».

#### Scenario: Неизвестный Zabbix
- **WHEN** `card_uuid` указывает на имя, которого нет в секции `zabbix`
- **THEN** сбор этого запроса завершается ошибкой с этим именем, а остальные запросы отчёта собираются как обычно

#### Scenario: Некорректный query
- **WHEN** `query` не является JSON-объектом или содержит ключ, отличный от `method` и `params`
- **THEN** сбор завершается ошибкой, запрос в Zabbix не отправляется

### Requirement: Только методы чтения
Коллектор MUST отправлять в Zabbix только методы вида `<сервис>.get`, где сервис состоит из строчных латинских букв, а точка в имени ровно одна. Любой другой метод MUST отклоняться ошибкой с именем метода до отправки запроса.

#### Scenario: Метод записи
- **WHEN** `query = {"method": "host.delete", "params": ["10084"]}`
- **THEN** сбор завершается ошибкой, содержащей `host.delete`, и в Zabbix не уходит ни одного запроса

#### Scenario: Метод записи под видом .get
- **WHEN** `query = {"method": "host.delete.get", "params": ["10084"]}`
- **THEN** сбор завершается ошибкой, содержащей `host.delete.get`, и в Zabbix не уходит ни одного запроса (Zabbix выполнил бы его как `host.delete`)

#### Scenario: Метод чтения
- **WHEN** `query = {"method": "problem.get", "params": {"severities": [4, 5]}}`
- **THEN** в Zabbix уходит вызов `problem.get` с этими params

### Requirement: Подстановка параметров
Плейсхолдеры `${name}` в строковых значениях `params` запроса SHALL заменяться значениями из params отчёта. Подставленное значение MUST оставаться частью той же JSON-строки и не может изменить структуру запроса. Плейсхолдер без значения MUST давать ошибку с его именем.

#### Scenario: Подстановка в строку
- **WHEN** `query.params = {"groupids": ["${group}"]}` и params = `{"group": "15"}`
- **THEN** в Zabbix уходит `{"groupids": ["15"]}`

#### Scenario: Кавычки в значении
- **WHEN** `query.params = {"search": {"name": "${q}"}}` и params = `{"q": "a\", \"x\": \"1"}`
- **THEN** в Zabbix уходит `search.name`, равный строке `a", "x": "1`, а новых ключей в запросе нет

#### Scenario: Нет значения
- **WHEN** в query есть `${host}`, а в params ключа `host` нет
- **THEN** сбор завершается ошибкой с именем `host`

### Requirement: Форма результата
Каждый элемент массива `result` SHALL давать одну строку. Вложенные объекты SHALL разворачиваться в колонки с ключами через точку, массивы объектов — с индексом (`hosts.0.host`). Скалярный результат или скалярный элемент массива SHALL давать строку `{result: <значение>}`. Значения SHALL передаваться так, как их вернул Zabbix, без перевода.

#### Scenario: Вложенные хосты и теги
- **WHEN** Zabbix возвращает `[{"eventid": "1", "severity": "4", "hosts": [{"host": "db1"}], "tags": [{"tag": "svc", "value": "api"}]}]`
- **THEN** результат — одна строка `{eventid: "1", severity: "4", hosts.0.host: "db1", tags.0.tag: "svc", tags.0.value: "api"}`

#### Scenario: countOutput
- **WHEN** Zabbix возвращает `"result": "42"`
- **THEN** результат — одна строка `{result: "42"}`

### Requirement: Ошибки ответа
Ответ Zabbix с объектом `error` MUST завершать запрос ошибкой с именем Zabbix, `message` и `data` из ответа. HTTP-статус вне 2xx или неразбираемое тело MUST завершать запрос ошибкой с HTTP-статусом и началом тела. Токен MUST не попадать в текст ошибки.

#### Scenario: Ошибка JSON-RPC
- **WHEN** Zabbix отвечает `{"error": {"code": -32602, "message": "Invalid params.", "data": "Invalid parameter \"/\": unexpected parameter \"foo\"."}}` на вызов `problem.get`
- **THEN** сбор завершается ошибкой, содержащей `Invalid params.` и `unexpected parameter`

#### Scenario: HTTP 502
- **WHEN** фронтенд Zabbix отвечает 502 с HTML-телом
- **THEN** сбор завершается ошибкой, содержащей `502`

### Requirement: Конфигурация источников
Секция конфига `zabbix` SHALL задавать именованные источники с полями `url` (корень фронтенда), `token` (API-токен, передаётся как Bearer) и `timeout` (по умолчанию 30s). Запись без `url` или `token` MUST останавливать старт с ошибкой, содержащей имя записи. Токен MUST маскироваться при логировании конфига.

#### Scenario: Пустой токен
- **WHEN** в секции `zabbix` есть запись `prod` без `token`
- **THEN** бот не стартует, ошибка содержит `prod`

#### Scenario: Логирование конфига
- **WHEN** конфиг с записью `zabbix.prod.token` выводится в лог
- **THEN** вместо токена в логе `***`

### Requirement: Вызов из Lua
Lua-скрипты SHALL вызывать коллектор как `collector.zabbix.collect(name, query, params)`, где `query` — JSON-строка того же формата, что поле `query`, с той же логикой, что у запроса `zabbix`. Функция SHALL быть доступна только при непустой секции `zabbix` и возвращать `rows, nil` или `nil, err`.

#### Scenario: Нет секции zabbix
- **WHEN** в конфиге нет секции `zabbix`
- **THEN** в Lua `collector.zabbix` равен nil
