# Proposal

## Why

Ревью `CODE_REVIEW_AND_DESIGN_2026-10-24.md` нашло три расхождения со спеками в уведомлениях и ошибках:
- ошибка пробного запуска уходит админу без замены секретов (`config-secrets`);
- подавленный дубль `skipped/duplicate` может прислать «отчёт снова работает», хотя спека `report-dedup` запрещает уведомлять о дубле. При этом сама `report-failure-alerts` считает любой `skipped` восстановлением — спеки противоречат друг другу;
- часть ошибок Grafana теряет HTTP-статус или начало тела ответа (`grafana-alerts-collector`).

## What Changes

- Текст ошибки в служебном сообщении пробного запуска проходит замену секретов.
- `skipped/duplicate` нейтрален для состояния уведомлений: не вызывает уведомлений и не считается «предыдущим запуском». После него восстановление определяется по последнему запуску, который не был дублем.
- Ошибки Grafana во всех ветках содержат HTTP-статус и до 200 символов тела ответа.

## Capabilities

### New Capabilities

### Modified Capabilities
- `report-failure-alerts`: из восстановления исключается `skipped/duplicate`, такой запуск не участвует в состоянии уведомлений.

## Impact

- Код: `internal/orchestrator/orchestrator.go` (`errText`), `internal/runlog/runlog.go`, `internal/db/queries/report_run.sql` (+ `task generate`), `internal/collector/grafana/collector.go`.
- Схема БД не меняется, меняется только запрос.
