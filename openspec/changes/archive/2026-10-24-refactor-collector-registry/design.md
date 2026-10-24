# Design

## Context

`collect` для Metabase/AppMetrica/JIRA вызывает `DataFetcher.Fetch(ctx, target, params)`, для остальных — `QueryFetcher.Fetch(ctx, target, crd.Query, params)`. Неизвестный тип даёт ошибку `unknown fetcher type`. Тесты передают `nil` для неиспользуемых источников.

## Goals / Non-Goals

**Goals:** выбор по типу в одном месте; добавление источника = одна строка регистрации.

**Non-Goals:** менять сигнатуры `Fetch` самих коллекторов (их зовёт и Lua stdlib); менять параллелизм, метрики, обработку ошибок `collect`.

## Decisions

1. **Реестр:** `type Fetchers map[string]QueryFetcher` (ключ — строковые константы `models.CollectType*`); `NewCollector(parallel uint8, fetchers Fetchers, log *slog.Logger)`. Альтернатива — регистрация методом `Register` после конструктора; отклонена: реестр неизменяем после сборки, map в конструкторе проще.
2. **Адаптер для источников без query:** экспортируемая `func IgnoreQuery(f DataFetcher) QueryFetcher` (тип-функция внутри). Нужна вне пакета — в `app` и `live-server`.
3. **Отсутствующий источник:** нет ключа в map (или значение `nil`) → ошибка `unknown fetcher type <type>`, как сейчас. В map не кладём nil-интерфейсы: проверка `f == nil` в `collect` страхует от паники.

## Risks / Trade-offs

- [typed-nil указатель коллектора в map даст панику при вызове] → конструкторы в `app` всегда возвращают не-nil; в `collect` проверяется только nil-интерфейс, что совпадает с текущим поведением (сейчас nil тоже паникует).
