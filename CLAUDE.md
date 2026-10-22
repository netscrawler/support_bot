# support_bot — правила работы Claude

Общаемся по-русски.

## Модели
- Основная сессия (Opus) — всё, где надо думать: требования, grilling, openspec propose/design, диагностика багов, ревью. Спорные решения — через `advisor`.
- Поиск и чтение кода — субагент `Explore` с `model: haiku` (внутри — `codegraph explore`).
- Написание кода по задачам из `tasks.md` — субагент `general-purpose` с `model: sonnet`.
- Промпт субагенту самодостаточен — он не видит разговора: задача из `tasks.md`, пути к файлам, критерий готовности, «ponytail full», «не править сгенерированный код».
- После каждой задачи Opus смотрит диф и запускает проверку (см. «Проект»). Галочка в `tasks.md` — только после этого.
- Задачи выполняются по очереди; параллельно — только если явно не трогают общие файлы.
- Эскалация: если Sonnet дважды не справился с задачей (build/тесты/линт красные), Opus делает её сам, при сомнениях — `advisor`.

## Процесс: openspec
- Любое изменение кода или поведения — через openspec: `/opsx:propose` → `/opsx:apply` → `/opsx:archive`.
- Без openspec можно: опечатки, комментарии, форматирование, бамп зависимостей.
- Перед `/opsx:propose`, если требования не заданы явно, — скилл `grilling`. При любой неясности по ходу работы — тоже grilling, не угадывать.
- Договорённости из grilling записываются в `design.md` изменения, а не остаются в чате.
- Баг: сначала первопричина через `diagnosing-bugs` (Opus), потом фикс.
- Коммит — после каждого `/opsx:archive`, в стиле репо: `feat(scope): описание` на русском. Push и деплой — только по команде.

## Качество
- Всегда `ponytail` (full): минимальное полное изменение. Требование передаётся и субагентам.
- Новая логика — с тестом, написанным до кода или вместе с ним (моки mockery, `pgxmock`).
- «Готово» / «тесты проходят» — только с выводом команды (`verification-before-completion`).
- После реализации всего изменения, перед `/opsx:archive` — агент `code-simplifier` на изменённых файлах, затем снова проверка.
- Изменения в `internal/processor/lua/` (Lua-песочница — граница безопасности) — `/security-review` перед archive.

## Проект
- Проверка: `go build ./... && go test ./... && golangci-lint run`.
- Нужен CGO (DuckDB): `CGO_ENABLED=1` и gcc, иначе build/test падают.
- Бинарники: `cmd/bot` (основной), `cmd/live-server`. Запуск: `task run` — читает `config/local.yaml` (в .gitignore; образец `config/config.example.yaml`, `config/example.env`).
- Генерируемый код не править: `internal/db/sqlcgen/` — менять `internal/db/queries/*.sql`, затем `task generate` (sqlc); `*/mock/mocks.go` — менять интерфейс, затем `go tool mockery`.
- Миграции `migrations/init.sql/` применяются только при первом старте пустой БД (`docker-entrypoint-initdb.d`). Старые файлы не править; схему менять новым `NNN_*.sql`, добавить его в `schema` в `sqlc.yaml` и отметить в `tasks.md`, что на существующей БД его надо накатить вручную.
- Без явной просьбы не трогать `deploy/ansible` и `.github/workflows`.

## Agent skills

### Issue tracker

Issues — локальные markdown-файлы в `.scratch/<feature>/`. См. `docs/agents/issue-tracker.md`.

### Triage labels

Стандартные: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. См. `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` и `docs/adr/` в корне. См. `docs/agents/domain.md`.
