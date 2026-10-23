# Design

## Context

- Админская «📤 Получить» (`AdminHandler.GenerateSelectedReport`) и пользовательская (`UserHandler.GenerateSelectedReport`) обе вызывают `tg_bot/service.Report.GenerateReportByName(ctx, name, chat)` → `ProduceSpecialEvent` → `SpecialEventForLK{EventTypeGenReportForTG, Recipient}`. Получатель подменяется на чат запросившего с `NeedDeleteAfterEndOfDay: false`, поэтому `sent_messages` уже не пишется.
- `generator.generate` при `!approve` возвращается без экспорта, так что обойти условие снаружи генератора нельзя.
- После `add-report-run-journal`: `processGenReportEvent` получает `kind`, `Recorder` не уведомляет по `test`/`lk` и учитывает в `LastStatus` только `cron`/`manual`.

Решения: grilling 2026-10-23, Q6 и Q23.

## Goals / Non-Goals

**Goals:** админ видит отчёт, результат условия и ошибку без похода в логи.

**Non-Goals:** новая кнопка, предпросмотр без отправки, пробный запуск в Max.

## Decisions

### D1. Признак пробного запуска в событии
`models.SpecialEventForLK` получает поле `Test bool`. Новый метод `EventAPI.ProduceTestEvent(ctx, name, recipient)` (то же, что `ProduceSpecialEvent`, но с `Test: true`) и `tg_bot/service.Report.GenerateTestReportByName(ctx, name, chat)`. `AdminHandler.GenerateSelectedReport` переходит на него. `UserHandler` не меняется. Альтернатива — новый `eventType` — отвергнута: событие то же, отличается только поведение. (агент)

### D2. Обход условия в генераторе
`models.Report` получает поле `IgnoreEvaluate bool` с тегом `json:"-"`, чтобы не попасть в `ctl export`/`apply`. В `generator.generate`: `if !approve && !report.IgnoreEvaluate { return data, nil, false, nil }`. В остальных случаях экспорт идёт как обычно, а наружу возвращается настоящий `approve`. Сигнатура `Generate` не меняется.

### D3. Оркестратор
В `run` для `EventTypeGenReportForTG`: если `event.Test`, то `kind = test` и `report.IgnoreEvaluate = true`. В `generateAndDeliver`, когда `kind == test`:
- после доставки шлёт в `recipientOverride` (`o.snd.Tg().SendText`) `🧪 evaluate = {approve}`; при false добавляет `— по расписанию отчёт не был бы отправлен`;
- при любой ошибке (генерация, отправка, пустой экспорт) шлёт туда же `❌ {html.EscapeString(err)[:500]}`;
- запись журнала: статус по доставке (`success`/`failed`/`partial`), при `approve=false` поле `reason = "evaluate_false"`. Статус `skipped` для `test` не используется, потому что отправка была.

Ошибка отправки служебных строк только пишется в лог.

### D4. Тексты
Сообщение после нажатия кнопки остаётся прежним («Отчёт поставлен в очередь…»), к нему добавляется « Это пробный запуск: условие отправки не проверяется».

## Risks / Trade-offs

- [`IgnoreEvaluate` — поле модели, которое не хранится в БД] → тег `json:"-"` и выставление только в оркестраторе на копии отчёта из `LoadByEvent`.
- [Отчёт без экспортов при пробном запуске] → админ получает только строку evaluate. Это корректная диагностика.
- [Служебные строки шлются только в Telegram] → `GenReportForTG` и так только Telegram.

## Migration Plan

Без миграций. Деплой обычным релизом после `add-report-run-journal`.
