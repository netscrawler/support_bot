# Proposal

## Why

Каждый новый HTTP-источник сейчас требует отдельного пакета коллектора (jira, appmetrica, grafana…), хотя большинство из них сводится к «GET → JSON/CSV → строки». Нужен универсальный коллектор, источник для которого можно описать прямо в отчёте в БД, не меняя конфиг и не выкатывая бота, а токены к нему хранить в БД зашифрованными.

## What Changes

- Новый тип запроса `http`: GET по URL из карточки или по имени источника из секции конфига `http_sources`, разбор ответа JSON / NDJSON / CSV в строки датасета.
- Таблица `queries` получает общие колонки `options jsonb` (настройки коллектора: заголовки, формат, путь до массива, тип авторизации) и `secret` (ссылка на секрет). `options` и `secret` проходят через JSON DSL отчётов (`ctl apply` / `ctl export`).
- Новая таблица `secrets`: значения шифруются AES-GCM мастер-ключом из `SECRETS_KEY`. Команды `ctl secret set|list|delete`.
- Коллектор `http` доступен в Lua (`collector.http.collect`) без доступа к секретам из БД.
- Новая миграция `011_query_options_secrets.sql` — на существующей БД накатывается вручную.

## Capabilities

### New Capabilities
- `http-collector`: запрос типа `http` — источник (конфиг или URL), авторизация, форматы ответа, форма строк, ошибки и лимиты.
- `db-secrets`: хранение секретов коллекторов в БД — шифрование, управление через `ctl`, маскирование значений.

### Modified Capabilities
<!-- нет: options/secret — новые поля карточки, поведение существующих коллекторов не меняется -->

## Impact

- Код: `internal/collector/` (новый пакет `http`, интерфейс `CardFetcher` в диспетчере), `internal/models/report.go`, `internal/store/` (report_store, новый secret_store), `internal/db/queries/` + `task generate`, `internal/pkg/secret/`, `internal/config/`, `internal/app/`, `internal/processor/lua/stdlib/collector.go`, `internal/service/report_validator.go`, `internal/cli/ctl.go`, `cmd/live-server/main.go`.
- БД: миграция 011 (новая таблица и две колонки с дефолтами — совместимо со старыми данными).
- Конфиг: новые секции `http_sources` и `secrets.key` / env `SECRETS_KEY`; без них бот стартует как раньше.
- Зависимостей не добавляется (`crypto/aes`, `encoding/csv` из stdlib).
