# Design

## Context

- `generator.Generate` возвращает `dataset` (`map[string][]map[string]any`) после pipeline, оркестратор получает его в `generateAndDeliver`.
- После `add-report-run-journal`: `report_runs`, `runlog.Recorder.Record`, интерфейс `orchestrator.RunRecorder`; `LastStatus` не учитывает `test`; статусы `success`/`skipped` не вызывают уведомлений о сбое.
- Поля отчёта читаются пятью запросами в `report.sql` (`GetReportRowByID`, `ListReportRows`, `ListActiveReportRows`, `GetActiveReportRowByName`, `GetAnyReportRowByName`) и пишутся через `CreateReport`.

Решения: grilling 2026-10-23, Q2, Q18, Q32 (подавляется только `cron`; `manual` шлёт всегда и становится точкой сравнения).

## Goals / Non-Goals

**Goals:** подавить точные повторы по cron одним флагом.

**Non-Goals:** дифф по строкам, временные окна, учёт по каждому получателю.

## Decisions

### D1. Миграция
`migrations/init.sql/009_report_dedup.sql`, добавить в `sqlc.yaml` после 007 и 008:
```sql
alter table reports add column dedup boolean not null default false;
alter table report_runs add column data_hash text;
create index report_runs_last_success on report_runs (report_name, started_at desc)
    where status = 'success' and kind in ('cron', 'manual');
```

### D2. Хэш датасета
`models.Dataset.Hash() string`: копия датасета без ключа `_meta` → `json.Marshal` → `sha256` → hex. `encoding/json` сортирует ключи map, так что порядок ключей не влияет, а порядок строк влияет. Значения, которые не маршалятся (теоретически из Lua), дают ошибку. Тогда хэш пустой и дедупликация для этого запуска не срабатывает (отчёт отправляется), а в лог пишется warn. Альтернатива — хэш экспортированных файлов — отвергнута: в xlsx и pdf есть метаданные со временем создания, и хэш менялся бы на каждом запуске. (агент)

### D3. Модель и хранение флага
`models.Report.Dedup bool` с тегом `json:"dedup,omitempty"`. Колонка `r.dedup` добавляется во все пять select-запросов и в `CreateReport`, маппинг в `report_store.go`. `report_example.go` (шаблон `ctl create`) получает `"dedup": false` с комментарием, если в шаблоне принято комментировать.

### D4. Проверка в оркестраторе
`RunRecorder` получает метод `LastSuccessHash(ctx, report) (string, bool)`. Запрос `LastReportRunSuccessHash :one`: `where report_name=$1 and status='success' and kind in ('cron','manual') order by started_at desc limit 1`. В `generateAndDeliver` после `approve == true`:
1. `hash := dataset.Hash()` всегда, кроме `test` и `lk`, и пишется в `Run.DataHash` (для `success` он станет точкой сравнения).
2. Если `report.Dedup && kind == cron && hash != ""` и `LastSuccessHash` вернул тот же хэш, отправки нет, `Run{Status: skipped, Reason: "duplicate"}`.

Ошибка чтения прошлого хэша → отчёт отправляется (лучше дубль, чем потеря) и warn в лог. (агент)

### D5. Уведомления
`skipped`/`duplicate` после `failed` по правилам runlog даёт «снова работает». Это корректно: запуск прошёл без ошибки. Отдельной логики нет.

## Risks / Trade-offs

- [Lua-скрипт кладёт в данные время генерации не в `_meta`] → дедупликация для такого отчёта не работает. В комментарии к флагу (`report_example.go`, README) указать: изменчивое время класть в `_meta`.
- [Числа из разных запусков приходят разными типами (int64/float64)] → для одного источника тип стабилен. Возможен лишний `success`, но не потеря отправки.
- [Повтор после `failed`: данные те же, что у прошлого `success`, а получатель прошлый `success` получил] → подавление корректно.

## Migration Plan

1. На существующей БД накатить вручную `009_report_dedup.sql` после `007` и `008`.
2. Включить `dedup` в JSON нужных отчётов и выполнить `ctl apply`.
3. Откат: старый бинарник колонки игнорирует.
