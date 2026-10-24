# Proposal

## Why

Ревью (`CODE_REVIEW_AND_DESIGN_2026-10-24.md`, Дизайн 2 и 4):
- `ReportStore.Load` пропускает отчёт, который не удалось собрать, и возвращает урезанный список с `nil`-ошибкой. Вызывающий (`ctl export`) молча выгружает неполный набор.
- `ReportStore.assemble` принимает девять отдельных значений, и каждое новое поле приходится протягивать через шесть одинаковых вызовов.

При проверке выяснилось, что `ReportStore.LoadActive` нигде не вызывается: расписание строится из `CronStore.LoadActive`. Тезис ревью о том, что отчёт «исчезает из расписания», не подтвердился, а метод остаётся мёртвым кодом.

## What Changes

- `ReportStore.Load` возвращает собранные отчёты и объединённую ошибку с именами несобранных отчётов.
- `ctl export` выгружает то, что собралось, и завершается ошибкой с перечнем несобранных отчётов.
- `assemble` принимает одну строку отчёта вместо девяти аргументов.
- Удаляется неиспользуемый `ReportStore.LoadActive` и строка интерфейса `orchestrator.ReportLoader`.

## Capabilities

### New Capabilities

### Modified Capabilities

Требования в спеках не меняются: для `ctl export` спеки нет, а остальное — внутренний рефакторинг. Поэтому `skip_specs: true`.

## Impact

- Код: `internal/store/report_store.go`, `internal/service/report_manager.go` (doc-комментарий), `internal/cli/ctl.go`, `internal/orchestrator/orchestrator.go` (интерфейс), мок `internal/orchestrator/mock/mocks.go` (перегенерация).
- `ctl export` теперь возвращает ненулевой код, если хотя бы один отчёт не собрался.
