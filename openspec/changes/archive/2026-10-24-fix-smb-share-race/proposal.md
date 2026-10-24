# Proposal

## Why

После `fix-delivery-failures` осталась гонка, отмеченная `shortcut:`: монитор SMB при переподключении переписывает `fs`/`session`/`conn`, а `Upload` одновременно читает `fs`. Это гонка данных: возможны паника или запись в закрытую шару.

## What Changes

- Доступ к соединению SMB (`fs`, `session`, `conn`) защищается мьютексом: монитор и `cleanup`/`connect` берут запись, `write` — чтение.
- Комментарий `shortcut:` удаляется.

## Capabilities

### New Capabilities

### Modified Capabilities

Наблюдаемое поведение не меняется (требование «Восстановление соединения SMB» уже есть в `delivery-retry`), поэтому `skip_specs: true`.

## Impact

`internal/delivery/smb/{client.go,reconn.go,client_test.go}`.
