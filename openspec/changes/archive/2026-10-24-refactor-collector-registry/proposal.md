# Proposal

## Why

Ревью (`CODE_REVIEW_AND_DESIGN_2026-10-24.md`, Дизайн 3): `collector.Collector` хранит семь отдельных полей-адаптеров, а `collect` повторяет `switch` по типу карточки. Каждый новый источник требует правки полей, конструктора, `switch` и обоих мест сборки (`internal/app/module_collectors.go`, `cmd/live-server/main.go`).

## What Changes

- Выбор источника по типу карточки — один реестр `тип → адаптер` внутри `Collector`; `switch` уходит.
- Все адаптеры приводятся к одному узкому интерфейсу `QueryFetcher`; источники без `query` (Metabase, AppMetrica, JIRA) оборачиваются маленьким адаптером.
- `NewCollector` принимает реестр; сборка в `app` и `live-server` передаёт его.

## Capabilities

### New Capabilities

### Modified Capabilities

Поведение не меняется (внутренний рефакторинг), поэтому `skip_specs: true`.

## Impact

`internal/collector/collector.go`, `internal/collector/collector_test.go`, `internal/app/module_collectors.go`, `cmd/live-server/main.go`.
