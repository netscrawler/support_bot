# Design

## Context

- `orchestrator.errText` только обрезает и экранирует текст, а `secret.Redact` вызывается позже, в `runlog.Record`, и только для журнала.
- `runlog.Record` читает прошлый статус через `LastReportRunStatus` (последний cron/manual) и считает любой статус, кроме `failed`/`partial`, восстановлением.
- `grafana.ruleRows` при `status != success` возвращает только `errorType: error`; `historyRows` при невалидном JSON не включает тело.

## Goals / Non-Goals

**Goals:** привести код к спекам `config-secrets`, `report-dedup`, `grafana-alerts-collector` и снять противоречие в `report-failure-alerts`.

**Non-Goals:** менять трактовку `skipped/evaluate` — он по-прежнему считается восстановлением (данные собрались, условие ложно), как и было в спеке.

## Decisions

1. **Нейтральный duplicate (решение пользователя).** Альтернатива «просто не уведомлять» оставляла дубль «предыдущим статусом», и после цепочки failed → duplicate → success уведомление о восстановлении не пришло бы никогда.
   - SQL `LastReportRunStatus`: `and (status <> 'skipped' or reason is distinct from 'duplicate')`. `reason` допускает NULL, поэтому `is distinct from`, а не `<>`. Других читателей у запроса нет.
   - `runlog.Record`: при `skipped` + `duplicate` выход до выбора уведомления (журнал и метрики пишутся как раньше). Без этой проверки сам дубль после `failed` прислал бы «снова работает».
   - Причина `duplicate` сейчас строковый литерал в оркестраторе; заводим константу в `models` рядом с `RunStatus*` и используем её в обоих местах.
2. **Секреты.** `errText` вызывает `secret.Redact(err.Error())` до обрезки: иначе обрезка может разрезать секрет и замена его не найдёт.
3. **Grafana.** `ruleRows` при `status != success` — `HTTP <code>: <errorType>: <error>: <начало тела>`; `historyRows` при ошибке разбора — добавить `%.200s` тела, как в `httpErr`. Токен в тело ответа не попадает (он в заголовке запроса), отдельной обработки не нужно.

## Risks / Trade-offs

- [Тест runlog использует fake store, а не SQL] → фильтр дубля дублируется в `fakeStore.LastStatus`; сам SQL проверяется тестом `run_store_test` по тексту запроса (pgxmock).
