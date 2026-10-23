# Design

## Context

- `generator.generate` для каждой карточки вызывает `Card.ResolveParams(ctx, eval)`: JSON `RawParams` → `map[string]string`, каждое значение через `Evaluator.EvalStr`. Значение с префиксом `=` компилируется как CEL с builtins (`today()`, `addDays`…) и пустой activation. Результат приводится к строке через `ResultString`. Коллекторы подставляют `Params` в запросы (`${name}` у pg/zabbix/prom и т.д.).
- `EvalStr` объявлен в трёх местах: реализация `evaluator.Engine`, интерфейсы `generator.Evaluator` и `models.Evaluator`. Других вызовов нет.
- Кнопки: админ «📤 Получить» — callback `report_get;{id};{name};{page}` (`CallbackDataContains("report_get")`), пользователь — `report_gen;…;{name}` (`CallbackDataContains("report_gen")`, имя в `data[2]`). Ограничение Telegram на callback data — 64 байта.
- После `improve-admin-test-run` админская кнопка идёт через `GenerateTestReportByName` → `ProduceTestEvent`.

Решения: grilling 2026-10-23, Q7, Q19, Q24, Q25, Q33 («▶️ Запустить» выбора периода не показывает, берёт `default_period`).

## Goals / Non-Goals

**Goals:** период через существующий механизм params, без изменения запросов и коллекторов.

**Non-Goals:** ввод произвольных дат, период для публичной ссылки, выбор часового пояса.

## Decisions

### D1. `models.Period`
`internal/models/period.go`: `type PeriodName string` с константами `PeriodToday`, `PeriodYesterday`, `PeriodWeek`, `PeriodMonth`; `func (p PeriodName) Range(now time.Time) (from, to time.Time, err error)` по правилам спеки (полночь через `time.Date(..., now.Location())`, как `today()` в builtins); `ParsePeriod(s string)` с `""` → `today`. Короткие коды для callback: `t`, `y`, `w`, `m` (`PeriodByCode`). Значение `week` = 7 полных дней до сегодня — решение агента: в Q7 договорились только про «неделю», без уточнений. Альтернативы («с понедельника» или «включая сегодня») отклонены как менее предсказуемые для cron-отчёта «за неделю». Трактовки `week` и `month` подтверждены в Q33.

### D2. CEL-переменные
В `evaluator.NewEngine` добавить `cel.Variable("period_from", cel.TimestampType)` и `cel.Variable("period_to", cel.TimestampType)`. Сигнатура меняется на `EvalStr(ctx, expr string, from, to time.Time) (string, error)` во всех трёх местах, activation — `{"period_from": from, "period_to": to}`. В `Evaluate` (условие отчёта) переменные не передаются: выражение условия с `period_*` упадёт на вычислении, и это ожидаемо. Альтернатива — передавать через `ctx` — отвергнута как неявная. (агент)

### D3. Проброс периода
- `models.Report.DefaultPeriod PeriodName` (`json:"default_period,omitempty"`, хранится в БД) и `models.Report.Period PeriodName` (`json:"-"`, выставляется на запуск).
- `Card.ResolveParams(ctx, eval, from, to)`. Генератор вычисляет `from, to := report.Period.Range(time.Now())`, пустой `Period` считается `DefaultPeriod`, а пустой `DefaultPeriod` — `today`.
- `models.SpecialEventForLK.Period PeriodName`. `ProduceSpecialEvent` и `ProduceTestEvent` получают аргумент периода. Оркестратор в `run` выставляет `report.Period = event.Period` для `lk` и `test`, для `cron` и `manual` оставляет пустым (тогда используется `DefaultPeriod`).
- `models.Report.UsesPeriod() bool`: в `RawParams` любой карточки есть подстрока `period_`.

### D4. Хранение и валидация
`migrations/init.sql/010_report_default_period.sql` (добавить в `sqlc.yaml`):
```sql
alter table reports add column default_period text not null default 'today';
```
Колонка добавляется во все select-запросы отчёта и в `CreateReport` (тот же набор, что в `add-report-dedup`). `service.ReportValidation` отклоняет значение не из четырёх. `report_example.go` получает `"default_period": "today"`.

### D5. Бот
- `tg_bot/service.Report.GetReportInfoByName` (и/или выборка для ЛК) возвращает `UsesPeriod` вместе с остальной информацией об отчёте.
- Админ: `report_get;{id};{name};{page}` без 5-го поля и с `UsesPeriod` → клавиатура из 4 кнопок `report_get;{id};{name};{page};{t|y|w|m}` и «Назад». С 5-м полем или без `UsesPeriod` → запуск. Пользователь — так же с `report_gen;…;{code}` в конце.
- Callback data не длиннее 64 байт: перед отрисовкой клавиатуры проверить длину. Если не помещается, запускать за `DefaultPeriod` и писать warn. Длинные имена отчётов уже сейчас рискуют упереться в этот лимит, поэтому так не становится хуже. (агент)
- Текст сообщения после запуска упоминает выбранный период: «Отчёт за вчера поставлен в очередь…».

## Risks / Trade-offs

- [Смена сигнатуры `EvalStr` затрагивает моки и тесты] → три места, компилятор найдёт все. Моки перегенерировать.
- [`week`/`month` трактуются не так, как ожидают пользователи] → трактовка показана в README и в подписи кнопки («Неделя (7 дн.)», «Прошлый месяц»).
- [Часовой пояс процесса] → тот же, что у `today()`, поведение не меняется.
- [Миграции 009 и 010 обе меняют `reports`] → независимы, порядок накатки по номеру.

## Migration Plan

1. На существующей БД накатить вручную `010_report_default_period.sql` после `007`–`009`.
2. Перевести нужные отчёты на `=period_from`/`=period_to` в params, задать `default_period`, выполнить `ctl apply`.
3. Откат: старый бинарник колонку игнорирует, но params с `period_*` в нём упадут на компиляции CEL, поэтому сначала вернуть params отчётов.
