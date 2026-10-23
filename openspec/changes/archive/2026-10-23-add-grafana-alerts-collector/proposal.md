# Proposal

## Why

В отчёты нужны данные об алертах Grafana: что горит прямо сейчас, в каком состоянии правила и что срабатывало за период. Сейчас их можно получить только через UI Grafana. Секция `grafana` в конфиге уже есть, она появилась вместе с коллектором Prometheus. Этому коллектору нужны только новые чтения через неё.

## What Changes

- Новый тип запроса `grafana_alerts`. В `card_uuid` хранится имя Grafana из секции `grafana`, в `queries.query` — режим (`active` | `rules` | `history`), в `params` — фильтры.
- `active`: экземпляры алертов (firing и pending) со значением, labels и annotations. Берутся из Prometheus-совместимого rules API Grafana.
- `rules`: одна строка на правило — состояние, health, время последней оценки, число активных экземпляров.
- `history`: смены состояний за период `from`/`to` из аннотаций Grafana (backend истории по умолчанию). `from`/`to` принимаются в RFC3339 или epoch ms, `limit` по умолчанию 10000.
- Фильтры `folder`, `group`, `state`, `alertname` применяются на нашей стороне. Неизвестный ключ params или неизвестный режим дают ошибку.
- Коллектор доступен из Lua: `collector.grafana_alerts.collect(name, mode, params)`.

## Capabilities

### New Capabilities
- `grafana-alerts-collector`: сбор данных отчёта об алертах Grafana-managed alerting (активные экземпляры, правила, история) из Grafana, описанной в конфиге, в том числе из Lua-скриптов.

### Modified Capabilities

## Impact

- Код: новый пакет `internal/collector/grafana`, ветка `grafana_alerts` в `internal/collector/collector.go`, константа в `internal/models/report.go`, `internal/app/module_collectors.go`, `internal/app/module_report.go` (Lua), `cmd/live-server/main.go`, `config/config.example.yaml` и комментарий секции `grafana` в `internal/config/config.go`.
- БД: без изменений.
- Безопасность: Lua-скрипты получат доступ на чтение к алертам всех Grafana из конфига. Перед archive обязателен `/security-review`.
- Зависимости: новых нет.
- Вне объёма: алерты из внешнего Alertmanager/Prometheus, история из Loki, информация о silence (её даёт только Alertmanager API).
