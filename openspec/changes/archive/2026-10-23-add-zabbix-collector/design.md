# Design

## Context

Механизм встраивания query-коллекторов уже есть благодаря `add-prometheus-collector` и `add-grafana-alerts-collector`. В него входят:
- интерфейс `collector.QueryFetcher.Fetch(ctx, target, query, params)` и ветка в switch `Collector.collect`;
- map `queries` в Lua-плагине (`collector.<name>.collect(target, query, params)`), куда модуль добавляется только при непустой секции конфига;
- маскирование секретов в `config.LogValue` с копированием map.

Zabbix 7.4 даёт JSON-RPC 2.0 по адресу `POST <frontend>/api_jsonrpc.php`. API-токен передаётся в заголовке `Authorization: Bearer`, поля `auth` в теле больше нет. Ошибки приходят с HTTP 200 в поле `error: {code, message, data}`.

Мотивация описана в proposal.md, требования — в `specs/zabbix-collector/spec.md`. Решения приняты в grilling 2026-10-23 (вопросы Q1–Q10 в сессии explore), решения агента помечены отдельно.

## Goals / Non-Goals

**Goals:**
- Один универсальный read-only вызов `*.get` вместо набора режимов (Q1, Q3, Q4).
- Встраивание в существующий `QueryFetcher`, switch и Lua `queries` без изменения общего кода.

**Non-Goals:**
- Готовые режимы с фильтрами. Их можно добавить позже поверх того же `Fetch`, если какие-то запросы будут повторяться (Q4).
- Перевод `severity`, `clock`, `value` в читаемый вид (Q6). Переводит Lua или шаблон.
- Автоматическая подстановка `limit` или `time_from` (Q8).
- Lua-таблица в качестве query (Q10).
- Вход по логину и паролю (`user.login`). Используется только API-токен (Q2, Q7).

## Decisions

### D1. Пакет и конфиг (Q2, Q7)
Новый пакет `internal/collector/zabbix`.
- `type Config struct{ URL, Token string; Timeout time.Duration }` с yaml/json-тегами и `comment`, как у `prometheus.GrafanaConfig`.
- `New(cfgs map[string]Config, log) (*Collector, error)`. Проверка выполняется здесь, как в `prometheus.buildSources`, потому что env-default для значений map не работает. Пустой `url` или `token` даёт `zabbix "<name>": не задан url|token`. Endpoint — `strings.TrimRight(url, "/") + "/api_jsonrpc.php"`, timeout — `cmp.Or(c.Timeout, 30s)`.
- `internal/config/config.go`: поле `Zabbix map[string]zabbix.Config` с yaml-ключом `zabbix`. В `LogValue` map копируется и `Token = "***"`.
- `cmd/live-server/main.go`: в его конфиге поле `Zabbix` с json-ключом `zabbix`, коллектор передаётся в `NewCollector`.

### D2. Хранение и встраивание (Q4, Q10)
- `models.CollectTypeZabbix = "zabbix"`, `card_uuid` хранит имя источника, `queries.query` — JSON. Миграции нет.
- `NewCollector` получает аргумент `zabbix QueryFetcher` после `alerts`, в switch добавляется ветка `c.zabbix.Fetch(ctx, target, crd.Query, params)`.
- Lua: `queries["zabbix"]` добавляется только при `len(cfg.Zabbix) > 0`. Коллектор создаётся всегда, поэтому решение принимается по конфигу, как для `prom` и `grafana_alerts` (так обходится ловушка typed nil).
- Новых интерфейсов нет, используется уже замоканный `collector.QueryFetcher`.

### D3. Разбор query и белый список (Q4)
- `json.Decoder` с `DisallowUnknownFields` и `UseNumber` разбирает query в `struct{ Method string; Params any }`. `UseNumber` нужен, чтобы `time_from: 1729700000` не превратился в float. Если остались лишние данные после объекта или `method` пуст, это ошибка.
- Метод должен совпадать с `^[a-z]+\.get$`, иначе ошибка `zabbix <name>: метод <m> запрещён, разрешены только *.get`. Проверка идёт до подстановки и до HTTP.
- Почему не `HasSuffix(".get")`: `/security-review` показал обход. `CJsonRpc.php` в Zabbix 7.4 делает `list($api, $method) = explode('.', $method)` и отбрасывает всё после второй точки. Поэтому `host.delete.get` проходит суффиксную проверку, а Zabbix выполняет `host.delete`. Ровно одна точка и только строчные буквы: Zabbix приводит имя к нижнему регистру, бот строже. Все сервисы API 7.4 называются строчными латинскими буквами.
- Альтернатива — явный список методов. Отклонена: шаблон покрывает все сервисы 7.4 и будущие тоже.

### D4. Подстановка (Q9)
Рекурсивный обход разобранного `Params` (map, slice, string). В каждой строке `${name}` заменяется regexp `\$\{(\w+)\}`, как в `prometheus/query.go`, но своим экземпляром, потому что тот не экспортирован, а выносить одну строку нет смысла. Ключи объектов не трогаются. После обхода запрос маршалится обратно, поэтому кавычки в значении экранирует `encoding/json`, и структура запроса не меняется. Если значения для плейсхолдера нет, возвращается ошибка `в params нет значения для ${name}`. Числовой литерал в query передаётся как есть. Подставленное значение всегда остаётся строкой, Zabbix принимает строки в числовых параметрах `.get` (риск ниже).

### D5. HTTP (решение агента)
- Тело `{"jsonrpc":"2.0","method":m,"params":p,"id":1}`, `Content-Type: application/json-rpc`, `Authorization: Bearer <token>`, `context.WithTimeout(ctx, src.timeout)`.
- Клиент — `&http.Client{}` **без** `retry.NewRoundTripper`. Этот транспорт повторяет тот же `*http.Request`, а при POST тело к повтору уже прочитано, и повтор ушёл бы с пустым телом. Кроме того, первая задержка 15s уже сравнима с timeout 30s. Чинить `retry` для одного потребителя не стоит. Если повторы понадобятся, сначала надо исправить сам `retry` (`req.GetBody`).
- Тело читается целиком. Если статус вне 2xx, ошибка `zabbix <name>: HTTP <code>: <первые 200 байт тела>`. Если тело 2xx не разбирается, ошибка со статусом. Если пришёл `error`, ошибка `zabbix <name>: <message>: <data>`. Токен есть только в заголовке.

### D6. Строки результата (Q5, Q6)
`result` разбирается в `any` обычным `json.Unmarshal`, без `UseNumber`.
- Массив: каждый элемент-объект проходит через `funcs.Flatten(obj)` и даёт строку. Скалярный элемент даёт `{result: v}`.
- Скаляр (`countOutput`) или объект (так отвечают некоторые `.get`, например `settings.get`): скаляр даёт одну строку `{result: v}`, объект — одну строку `funcs.Flatten(obj)`.
- Пустой массив даёт пустой результат без ошибки.
- Почему без `UseNumber`: `luapkg.GoValueToLua` не знает `json.Number`, а `float64` понимает. Zabbix почти всё отдаёт строками, поэтому редкие числа в `float64` ничего не теряют. В запросе `UseNumber` остаётся (D3): там `json.Number` маршалится обратно литералом.

### D7. Тесты (решение агента)
- `internal/collector/zabbix`, `httptest.Server`:
  - путь `/api_jsonrpc.php`, заголовок Bearer, Content-Type, `jsonrpc/method/params` в теле;
  - не-`.get` метод, `host.delete.get`, `.get`, `host..get`, `HOST.GET` и лишний ключ query — без запросов к серверу (счётчик запросов равен 0);
  - подстановка, кавычки в значении, отсутствующий плейсхолдер, числовой литерал сохраняется (`1729700000`, не `1.7297e+09`);
  - Flatten вложенных `hosts`/`tags`, скаляр `countOutput`, объектный result, пустой массив, число в result приходит как `float64`;
  - JSON-RPC error, HTTP 502 с HTML;
  - неизвестный источник; `New` с пустым `url`/`token`.
- `collector_test.go`: ветка `zabbix` вызывает мок `QueryFetcher` с `(CardUUID, Query, Params)`.
- `config_test.go`: токен zabbix маскируется в `LogValue`, исходный map не меняется.
- Проверка «без секции нет модуля» находится в провайдере `module_report.go`, тестов у `internal/app` нет. Она проверяется ручным прогоном.

## Risks / Trade-offs

- [Lua и запросы видят всё, что разрешено токену, включая `user.get`, `token.get`, `usermacro.get` с несекретными макросами] → токен выдаётся отдельному пользователю. Его роль: в API allow list только `*.get` (маски поддерживаются в 7.4), доступ к хостам — read-only через группы. Это описывается в `config.example.yaml`. Перед archive обязателен `/security-review`.
- [Без `limit` метод `history.get` может вернуть огромный ответ, и тело целиком окажется в памяти бота] → память не защищена. Timeout ограничивает только время, а быстрый Zabbix успеет отдать много данных. За размер отвечает автор запроса (Q8), в примере конфига показаны `limit` и `time_from`. Если это станет проблемой, отдельным изменением добавить лимит тела (`io.LimitReader` с ошибкой при превышении).
- [Подставленные значения всегда строки, а какой-то параметр `.get` может не принять строку вместо числа] → для известных параметров (`limit`, `time_from`, `time_till`, `severities`, `groupids`) Zabbix принимает строки. Если найдётся исключение, отдельным изменением добавить подстановку числом.
- [Колонки с индексами (`hosts.0.host`) неудобны для массивов переменной длины] → принимаем (Q5). Для сложной формы данных есть Lua.
- [Нет повторов при сетевых ошибках] → отчёт по расписанию запустится в следующий раз. Ошибка одного запроса не роняет остальные (см. `Collector.Collect`).

## Migration Plan

1. В Zabbix создать пользователя с ролью, у которой в API allow list `*.get`, и группой с read-only правами на нужные хосты. Выпустить API-токен.
2. Добавить в рабочий конфиг секцию `zabbix: {<имя>: {url, token}}`.
3. Завести запросы `q_type = 'zabbix'`, миграции БД нет.
4. Откат: старый бинарник ответит `unknown fetcher type` на этот тип, остальные запросы не затрагиваются. Лишнюю секцию `zabbix` в конфиге старый бинарник игнорирует: `cleanenv.ReadConfig` неизвестные ключи не отвергает.
