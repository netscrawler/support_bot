# http-collector Specification

## Purpose
Позволяет собирать в отчёты данные из любого HTTP API, который отдаёт JSON, NDJSON, CSV, XML, XLSX, YAML или текст: источник описывается в конфиге бота или прямо в карточке отчёта в БД.

## Requirements

### Requirement: Источник запроса типа http
Для запроса с типом `http`, если `card_uuid` совпадает с именем записи в секции конфига `http_sources`, URL SHALL строиться как `base_url` этой записи + `query` (путь), а авторизация, заголовки и таймаут SHALL браться из этой записи. Иначе `card_uuid` MUST быть абсолютным `http://` или `https://` URL и используется как есть. Запрос SHALL выполняться методом GET; params запроса SHALL добавляться в query string.

#### Scenario: Источник из конфига
- **WHEN** в конфиге `http_sources.billing.base_url = https://billing.local`, карточка `card_uuid = billing`, `query = /api/items`, params `{"from": "2026-10-01"}`
- **THEN** уходит `GET https://billing.local/api/items?from=2026-10-01` с авторизацией из `http_sources.billing`

#### Scenario: URL из БД
- **WHEN** карточка `card_uuid = https://api.example.com/v1/stats`, имени `https://api.example.com/v1/stats` в конфиге нет
- **THEN** уходит `GET https://api.example.com/v1/stats`

#### Scenario: Не URL и не имя из конфига
- **WHEN** `card_uuid = billingg`, такой записи в `http_sources` нет
- **THEN** сбор этого запроса завершается ошибкой с этим значением, запрос не отправляется, остальные запросы отчёта собираются как обычно

### Requirement: Опции запроса
Колонка `options` запроса SHALL задавать для `http`: `headers` (объект строк), `format`, `root`, `delimiter`, `sheet`, `auth` (`bearer`, `basic` или `header`), `auth_header`, `timeout`. Опции из карточки SHALL перекрывать значения из конфига; заголовки SHALL объединяться с перекрытием по имени. Неизвестный ключ или значение `format`/`auth` вне списка MUST отклоняться при валидации отчёта.

#### Scenario: Перекрытие заголовка
- **WHEN** в конфиге `headers: {Accept: text/csv}`, в options `{"headers": {"Accept": "application/json"}}`
- **THEN** запрос уходит с `Accept: application/json`

#### Scenario: Неизвестная опция
- **WHEN** отчёт применяется через `ctl apply` с карточкой `http` и options `{"fromat": "csv"}`
- **THEN** валидация отчёта завершается ошибкой с именем `fromat`, отчёт не сохраняется

### Requirement: Авторизация из секрета
Если у карточки задан `secret`, его расшифрованное значение SHALL использоваться для авторизации по `options.auth`: `bearer` — `Authorization: Bearer <значение>`, `basic` — значение в формате `user:password`, `header` — заголовок `options.auth_header` со значением. Секрет карточки SHALL перекрывать авторизацию из конфига. Если `secret` задан, а `auth` нет, SHALL использоваться `bearer`.

#### Scenario: Bearer из секрета
- **WHEN** карточка `secret = billing_token` (значение `t0k`), options без `auth`
- **THEN** запрос уходит с `Authorization: Bearer t0k`

#### Scenario: Ключ в своём заголовке
- **WHEN** options `{"auth": "header", "auth_header": "X-Api-Key"}`, секрет `k3y`
- **THEN** запрос уходит с `X-Api-Key: k3y` и без `Authorization`

#### Scenario: Секрет не найден
- **WHEN** карточка ссылается на секрет, которого нет, или `SECRETS_KEY` не задан
- **THEN** сбор этого запроса завершается ошибкой с именем секрета, запрос не отправляется

### Requirement: Формат ответа
Формат ответа SHALL определяться по `options.format` (`json`, `ndjson`, `csv`, `xml`, `xlsx`, `yaml`, `text`), а если он не задан — по `Content-Type` по таблице из сценария «Соответствие Content-Type». Неизвестный формат MUST давать ошибку с полученным `Content-Type`.

#### Scenario: Соответствие Content-Type
- **WHEN** `format` не задан, а `Content-Type` ответа — `application/json` / `application/x-ndjson` / `text/csv` или `application/csv` / `application/xml` или `text/xml` / `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` / `application/yaml`, `application/x-yaml` или `text/yaml` / `text/plain`
- **THEN** тело разбирается как json / ndjson / csv / xml / xlsx / yaml / text соответственно

#### Scenario: Формат из Content-Type
- **WHEN** ответ `Content-Type: text/csv; charset=utf-8`, `format` не задан
- **THEN** тело разбирается как CSV

#### Scenario: Формат перекрыт
- **WHEN** ответ `Content-Type: application/octet-stream`, `options.format = json`
- **THEN** тело разбирается как JSON

#### Scenario: Неизвестный тип
- **WHEN** ответ `Content-Type: text/html`, `format` не задан
- **THEN** сбор завершается ошибкой, содержащей `text/html`

### Requirement: Форма строк из JSON
Для json SHALL браться значение по `options.root` (ключи через точку; без `root` — весь документ). Массив SHALL давать по строке на элемент, объект — одну строку. Вложенные объекты SHALL разворачиваться в колонки через точку, массивы объектов — с индексом, как у коллектора zabbix. Скаляр SHALL давать строку `{value: <скаляр>}`. Отсутствующий `root` MUST давать ошибку с путём.

#### Scenario: Массив по root
- **WHEN** ответ `{"data": {"items": [{"id": 1, "user": {"name": "a"}}]}}`, `root = data.items`
- **THEN** результат — одна строка `{id: 1, user.name: "a"}`

#### Scenario: Массив скаляров
- **WHEN** ответ `[1, 2]`
- **THEN** результат — строки `{value: 1}` и `{value: 2}`

#### Scenario: Нет root
- **WHEN** `root = data.rows`, в ответе нет ключа `rows`
- **THEN** сбор завершается ошибкой, содержащей `data.rows`

### Requirement: Форма строк из NDJSON и CSV
Для ndjson каждая непустая строка тела SHALL разбираться как JSON-объект и давать одну строку результата по тем же правилам. Для csv первая строка SHALL быть заголовком с именами колонок, BOM в начале SHALL отбрасываться, разделитель — `options.delimiter` (по умолчанию `,`), значения SHALL оставаться строками. Строка с другим числом полей MUST давать ошибку с номером строки.

#### Scenario: CSV с BOM и точкой с запятой
- **WHEN** тело `﻿date;count\n2026-10-01;5\n`, `delimiter = ;`
- **THEN** результат — одна строка `{date: "2026-10-01", count: "5"}`

#### Scenario: NDJSON
- **WHEN** тело `{"a":1}\n\n{"a":2}\n`
- **THEN** результат — строки `{a: 1}` и `{a: 2}`

### Requirement: Форма строк из YAML
YAML SHALL разбираться по тем же правилам, что JSON: `root`, массив → строки, объект → одна строка с развёрнутыми вложенными объектами, скаляр → `{value: ...}`. Берётся первый документ.

#### Scenario: YAML-список
- **WHEN** тело `items:\n  - id: 1\n    tags: {env: prod}\n`, `root = items`
- **THEN** результат — одна строка `{id: 1, tags.env: "prod"}`

### Requirement: Строки из XML
Для xml `root` SHALL задавать путь имён элементов от корневого элемента документа (`rss.channel.item`); каждый элемент, совпавший с полным путём, SHALL давать одну строку. Без `root` строками SHALL быть дочерние элементы корневого. Префиксы пространств имён SHALL отбрасываться. Путь `root`, не найденный в документе, MUST давать ошибку с путём.

#### Scenario: Нет root в XML
- **WHEN** `root = rss.feed.item`, в документе нет `feed`
- **THEN** сбор завершается ошибкой, содержащей `rss.feed.item`

### Requirement: Колонки из XML-элемента
Атрибуты элемента строки SHALL давать колонки `@<имя>`, дочерние элементы — колонки по имени (вложенные — через точку, повторяющиеся — с индексом), текст элемента без дочерних и атрибутов — значение колонки, текст элемента с дочерними или атрибутами — колонка `#text`. Все значения — строки.

#### Scenario: RSS
- **WHEN** тело `<rss><channel><item id="1"><title>A</title></item><item id="2"><title>B</title></item></channel></rss>`, `root = rss.channel.item`
- **THEN** результат — строки `{@id: "1", title: "A"}` и `{@id: "2", title: "B"}`

#### Scenario: Повторяющиеся дочерние
- **WHEN** элемент строки `<row><tag>a</tag><tag>b</tag></row>`
- **THEN** строка `{tag.0: "a", tag.1: "b"}`

### Requirement: Форма строк из XLSX и текста
Для xlsx SHALL читаться лист `options.sheet`, а без него — первый лист; первая строка — заголовок, значения — строки в том виде, в каком их показывает Excel; пустые строки пропускаются. Несуществующий лист MUST давать ошибку с его именем. Для text каждая непустая строка тела SHALL давать строку `{line: <текст строки без \r>}`.

#### Scenario: Лист по имени
- **WHEN** ответ — xlsx с листами `Summary` и `Data`, `options.sheet = Data`, в `Data` строки `date,count` и `2026-10-01,5`
- **THEN** результат — одна строка `{date: "2026-10-01", count: "5"}`

#### Scenario: Нет листа
- **WHEN** `options.sheet = Nope`, такого листа нет
- **THEN** сбор завершается ошибкой, содержащей `Nope`

#### Scenario: Текст
- **WHEN** тело `a\r\n\nb\n`, `format = text`
- **THEN** результат — строки `{line: "a"}` и `{line: "b"}`

### Requirement: Ошибки и лимиты ответа
Ответ с кодом вне 2xx MUST давать ошибку с кодом и не более чем первыми 512 байтами тела. Тело больше 50 МБ MUST давать ошибку, а не обрезанный результат. Редирект на другой хост или со сменой схемы MUST не выполняться. Разбор MUST давать ошибку, а не ронять процесс, на XML глубже 1000 уровней и на xlsx, распакованный размер которого больше 50 МБ. Значение секрета MUST не попадать в текст ошибки и в лог, даже если сервер вернул его в теле.

#### Scenario: 500 от API
- **WHEN** API отвечает `500` с телом `internal error`
- **THEN** сбор завершается ошибкой, содержащей `500` и `internal error`

#### Scenario: Токен в теле ошибки
- **WHEN** API отвечает `401` с телом `bad token t0k`, а `t0k` — значение секрета карточки
- **THEN** в тексте ошибки и в логе вместо `t0k` стоит `***`

#### Scenario: Слишком большой ответ
- **WHEN** тело ответа больше 50 МБ
- **THEN** сбор завершается ошибкой о превышении лимита, строки не возвращаются

#### Scenario: Редирект https→http
- **WHEN** источник `https://api.example.com` отвечает `302` на `http://api.example.com/x`
- **THEN** редирект не выполняется, сбор завершается ошибкой со статусом `302`, секрет не уходит по http

#### Scenario: Глубокий XML
- **WHEN** тело — XML с 100000 вложенных элементов
- **THEN** сбор завершается ошибкой о глубине, процесс бота продолжает работать

#### Scenario: xlsx-бомба
- **WHEN** тело — xlsx меньше 50 МБ, распакованный размер которого больше 50 МБ
- **THEN** сбор завершается ошибкой, процесс бота продолжает работать

### Requirement: http в Lua
Lua-скрипты SHALL вызывать коллектор как `collector.http.collect(target, query, params, options)`, где `options` — необязательная таблица с теми же ключами, что колонка `options`. Секреты из БД MUST быть недоступны из Lua: ключ `secret` в `options` MUST давать ошибку.

#### Scenario: Вызов из Lua
- **WHEN** скрипт вызывает `collector.http.collect("billing", "/api/items", {}, {root = "data"})`
- **THEN** скрипт получает строки из `data` ответа источника `billing`

#### Scenario: Секрет из Lua
- **WHEN** скрипт передаёт `options = {secret = "billing_token"}`
- **THEN** вызов возвращает `nil, <ошибка>`, запрос не отправляется
