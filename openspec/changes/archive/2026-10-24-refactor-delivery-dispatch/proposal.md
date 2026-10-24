# Proposal

## Why

Ревью (`CODE_REVIEW_AND_DESIGN_2026-10-24.md`, Дизайн 5): `models.Message.Send` переключает четыре типа получателя, `models.SenderProvider` выставляет четыре метода, а `models` зависит от формата письма `smtp.Mail`. Любое изменение канала доставки затрагивает доменную модель.

## What Changes

- Диспетчеризация доставки (`Send`, отправка в Telegram/SMTP/SMB/Max, сборка письма) переезжает из `models` в новый пакет `internal/delivery`.
- Интерфейсы отправителей и `SenderProvider` уходят из `models`; `models` больше не импортирует `internal/delivery/smtp`.
- Оркестратор зависит от узкого интерфейса отправки сообщения и отдельного узкого интерфейса служебной строки пробного запуска.

## Capabilities

### New Capabilities

### Modified Capabilities

Поведение доставки не меняется (включая частичную доставку Max из `fix-delivery-failures`), поэтому `skip_specs: true`.

## Impact

`internal/models/{message.go,send.go,message_test.go}`, новый `internal/delivery/{dispatch.go,dispatch_test.go}`, `internal/orchestrator/{orchestrator.go,orchestrator_test.go}`, `internal/app/{module_delivery.go,module_report.go}`.
