# Архитектурный аудит — support_bot (2026-09-15, ветка `api`)

Критический разбор архитектуры, без похвалы. ~21k строк Go, часть кода в процессе
рефакторинга (унификация generator/orchestrator, добавление HTTP API).

## 1. Composition root — грязь

`internal/app/app.go`, функция `init` (~220 строк, app.go:204-416) вручную собирает
~25 компонентов (БД, 2 бота, 4 канала доставки, коллектор, генератор, оркестратор,
шедулер, lua-manager, HTTP-сервер) без группировки, без под-конструкторов, без фаз
запуска. Каждый новый компонент — ещё десяток строк в ту же функцию.

## 2. `max_bot` — заглушка, а не второй бот

54 строки (`internal/max_bot/bot.go`), нет handlers/menu/service/repository — у
`tg_bot` есть всё это. Пользователи MAX могут только получать пуши
(`internal/delivery/max/max.go`), никакого взаимодействия с ботом. По коду
расползлись `if ChType == Max` (deleter, sender provider, admin-хендлеры) вместо
одной точки ветвления.

## 3. Lua-песочница — «безопасность через уборку», не через конструкцию

`internal/processor/lua/runtime.go:205-206` сначала грузит `io` и `debug`
(`lua.OpenIo`, `lua.OpenDebug`), затем `removeDangerousFunctions()` (строка 210,
рядом `TODO` автора о недоделанности) пытается их вычистить. Один пропущенный шаг —
и скрипт получает файловый ввод-вывод и интроспекцию VM.

Дополнительно:
- `http`/`url` разрешены по умолчанию — SSRF без allowlist адресов.
- `duck.query(sql)` (`stdlib/database.go:90-110`) пускает произвольный SQL прямо из
  Lua-скрипта.
- Сейчас ограничено тем, что `ScriptManager` вызывается только из CLI
  (`internal/cli/ctl.go:335`), не из бота — но это не гарантия кодом, а то, что
  пока никто не подключил.

## 4. Утечка секретов в логи — структурный риск

`internal/cli/run.go:46` логирует весь конфиг целиком. Спасает только вручную
поддерживаемый список редактируемых полей в `Config.LogValue()`
(config.go:107-118, 7 полей). Новый secret в одном из 9 вложенных конфигов, про
который забыли дописать в `LogValue()`, — пароль в логах.

`Validate()` — по сути no-op (`// TODO: add full config validation`,
config.go:82-84): проверяются только Log и HTTP, остальное (DB/SMTP/SMB/Jira/
AppMetrica/боты) падает где-то в рантайме, не при старте.

## 5. Двойная система ошибок

`internal/errorz` и `internal/models` независимо объявляют свои `ErrNotFound`/
`ErrInternal`/`ErrAlreadyExist` с разным текстом `.Error()`.
`internal/repository/report.go` мешает обе таксономии в одном файле, а
`service/report_generator.go` вручную транслирует `models.ErrNotFound` →
`errorz.ErrNotFound` только чтобы HTTP-хендлер сработал через `errors.Is`.
Следующий «not found»-путь про этот перевод легко забудут.

## 6. Реальный баг

`internal/orchestrator/result_repo.go:132-140`: если `BeginTxx` падает, код
логирует, откатывается на прямой `ExecContext`, но затем всё равно доходит до
`defer tx.Rollback()` и `tx.ExecContext(...)` на nil-транзакции из проваленного
`BeginTxx`. Это сломанная ветка кода, не стилистика.

## 7. Конкурентность без берегов

`Orchestrator.processGenReportEvent` плодит `go o.generateAndDeliver(...)` без
ограничения (orchestrator.go:728-763), хотя внутри генератора всего 4 воркера
(app.go:332). При всплеске событий — неограниченный рост горутин, каждая держит
контекст и репорт в замыкании.

Отдельно: у воркера генератора собственный таймаут 5 минут (generator.go:222)
независимо от дедлайна вызывающего — HTTP-запрос с таймаутом 30с давно отвалился,
а воркер всё ещё занят до 5 минут.

## 8. Тесты покрывают только то, что трогали в этом рефакторинге

20 test-файлов на 56 пакетов. Ноль тестов: `internal/app`, `internal/repository`
(основной SQL-слой отчётов), весь `internal/tg_bot/*` (хендлеры, сервис,
репозиторий), `internal/max_bot`, весь `internal/delivery/*` (smtp/smb/telegram/
max), `internal/sheduler`, `internal/exporter/*` кроме text.

## Мелочи

- `internal/sheduler` — опечатка вшита в публичный импорт-путь навсегда.
- `collector.ErrEmtyCard` — опечатка, grep на "empty" не найдёт.
- Два живых источника миграций: `db/schema.sql` (не трогали с 2026-08-06, похоже
  на мёртвый артефакт) и `migrations/init.sql/*.sql` (актуальный, 4 файла).
- `internal/pkg` — свалка (`converter.go`, `escape.go`, `font.go`,
  `generate_env.go`, `proxy.go`, `struct_to_yaml.go` без общей темы) рядом с
  нормально оформленными подпакетами.

## Что НЕ проблема, вопреки виду

`go.mod` выглядит раздутым (AWS SDK v2, весь GCP-стек, charmbracelet/bubbletea),
но это тянется исключительно через `tool github.com/go-task/task/v3/cmd/task` —
dev-инструмент, не прикладной код. Проверено grep'ом по internal/ и cmd/.

## Итог по over-engineering

Склеить `max_bot`+`delivery/max` в одну зону ответственности вместо
`if ChType==Max` по всему коду, убрать один из наборов ошибок (errorz/models),
разбить `app.init()` на 4-5 конструкторов по фазам. Не «удалить N строк» — это
структурный долг: короче код не станет, но перестанет расползаться.
