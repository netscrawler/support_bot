# Proposal

## Why

На старте весь конфиг пишется в лог (`internal/cli/run.go`), и от утечки паролей защищает только список полей в `Config.LogValue()`, который ведётся руками. Новый токен, который забыли туда добавить, окажется в логах (аудит 2026-09-15, п. 4). `Config.Validate()` проверяет только `log` и `http`, поэтому неполный конфиг SMTP, SMB или Max всплывает ночью на первом cron-запуске, а не при старте.

## What Changes

- Новый тип `secret.String` для всех паролей, токенов и DSN в конфигах. В логах, `fmt` и JSON он показывается как `***`, а в коде, который им пользуется, берётся через `.Value()`. Ручной список маскирования в `Config.LogValue()` удаляется.
- Значения всех секретов конфига регистрируются при загрузке, а `secret.Redact(text)` заменяет их на `***` в произвольном тексте. Так секреты вычищаются из текстов ошибок, которые уходят в журнал запусков и в уведомления админам.
- `Config.Validate()` проверяет обязательные поля каждой включённой секции (включена — значит, задано её ключевое поле: host, url, enabled). При любой ошибке бот не стартует, все ошибки выводятся разом с путями полей.
- `config validate` в CLI использует ту же проверку.

## Capabilities

### New Capabilities
- `config-secrets`: секреты конфига не попадают в логи и в тексты ошибок, которые видят пользователи.
- `config-validation`: проверка полноты конфига при старте.

### Modified Capabilities

## Impact

- Код: новый пакет `internal/pkg/secret`. Конфиги `internal/postgres`, `internal/tg_bot`, `internal/max_bot`, `internal/delivery/smtp`, `internal/delivery/smb`, `internal/collector/{jira,appmetrica,postgres,prometheus,zabbix}`: тип поля и места использования (`.Value()`). `internal/config/config.go` (`LogValue`, `Validate`, регистрация секретов), `internal/runlog` (`Redact` перед записью ошибки и уведомлением), `cmd/live-server/main.go`.
- Совместимость: формат YAML и env не меняется. `config create`/`config update` пишут реальные значения, как и раньше.
- Зависит от `add-report-run-journal` (место применения `Redact`).
- Вне объёма: хранение секретов во внешнем vault, шифрование конфига, маскирование секретов в данных отчётов.
