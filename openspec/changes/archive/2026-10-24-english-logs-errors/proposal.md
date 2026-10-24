# Proposal

## Why

Логи и тексты ошибок написаны вперемешку на русском и английском. Это мешает искать по логам и заводить алерты на сообщения. Служебный язык — английский; на русском остаётся только то, что видит пользователь в Telegram.

## What Changes

- Тексты ошибок (`fmt.Errorf`, `errors.New`) с кириллицей переводятся на английский: коллекторы grafana/prometheus/zabbix, валидация конфига prometheus, `internal/config`, экспортёр xlsx, `internal/cli/link.go`.
- Сообщения логов (`slog` `*Context`) с кириллицей переводятся на английский: `internal/orchestrator`, `internal/runlog`.
- Тесты, которые проверяют эти строки, обновляются; сообщения `t.Fatal`/`t.Errorf` и тестовые ошибки-фикстуры с кириллицей тоже переводятся.
- Ключи логов, коды и структура сообщений (префиксы `zabbix %q:`, `runlog:` и т.п.) не меняются — меняется только язык.

## Capabilities

### New Capabilities

Нет.

### Modified Capabilities

Нет — спеки не фиксируют точные тексты логов и ошибок (`skip_specs: true`).

## Impact

- Код: ~60 строк в `internal/collector/{grafana,prometheus,zabbix}`, `internal/config`, `internal/exporter/xlsx`, `internal/cli/link.go`, `internal/orchestrator`, `internal/runlog` и тесты к ним.
- Внешнее поведение: меняется текст ошибок в выводе CLI `ctl link revoke` и в ошибках, которые доходят до администраторов (например, в алертах о сбое отчёта текст ошибки будет на английском, сама обвязка алерта — на русском).
- API, конфиг, БД, зависимости — без изменений.
