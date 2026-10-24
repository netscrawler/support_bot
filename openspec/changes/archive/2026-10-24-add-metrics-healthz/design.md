# Design

## Context

- Публичный сервер `internal/api/http.Server` поднимается в `app/module_http.go`, конфиг `http.host/port`. Он смотрит наружу (Q8).
- `internal/postgres.DB.GetConn()` отдаёт `*pgxpool.Pool`.
- После предыдущих изменений: каждая запись журнала проходит через `runlog.Recorder.Record`, а каждая отправка — через `retry.Do(ctx, delays, log, op, fn)`.
- `collector.Collector.collect` запускает горутину на карточку и выбирает `fetchFn` по `crd.Type`.
- `sheduler.Sheduler` имеет `Start`/`Stop`, админка вызывает их через `StartJobs`/`StopJobs`.
- `github.com/prometheus/client_golang v1.12.1` уже есть в `go.mod` как indirect.

Решения: grilling 2026-10-23, Q8, Q20.

## Goals / Non-Goals

**Goals:** минимальный набор метрик, который отвечает на вопросы «жив ли бот», «падают ли отчёты», «что тормозит».

**Non-Goals:** трейсинг, метрики по каждому отчёту (метка `report` не вводится из-за кардинальности), отдельная readiness.

## Decisions

### D1. Пакет `internal/metrics`
Глобальные метрики на `prometheus.DefaultRegisterer` через `promauto` в одном файле `metrics.go`: `ReportRuns` (CounterVec kind,status), `ReportRunDuration` (HistogramVec kind, бакеты 0.5…600s), `CollectorDuration` (HistogramVec type), `CollectorErrors` (CounterVec type), `DeliveryRetries` и `DeliveryFailures` (CounterVec op), `SchedulerRunning` (Gauge), `BuildInfo` (GaugeVec version,commit). Глобальные переменные, а не DI, потому что метрики пишутся в 4 пакетах, а прокидывать регистр через fx в каждый — лишний код без пользы. Тесты читают значения через `testutil.ToFloat64`. (агент)

### D2. Сервер
`metrics.Server` в том же пакете: `http.Server` с `ReadHeaderTimeout 5s` и mux на `/metrics` (`promhttp.Handler()`) и `/healthz`. Healthz получает функцию `ping func(ctx) error` (в app: `db.GetConn().Ping`) и вызывает её с `context.WithTimeout(r.Context(), 2s)`. fx-провайдер в `internal/app/module_http.go`: при `cfg.Metrics.Enabled == false` сервер не создаётся и хуки не регистрируются. Start/Shutdown по образцу `apihttp.Server`. Ошибка `ListenAndServe`, кроме `ErrServerClosed`, пишется в лог.

### D3. Healthz без планировщика (пересмотр Q20, подтверждено в Q30)
В Q20 согласовали «ping PostgreSQL плюс жив ли scheduler». Но админ может остановить задачи кнопкой («StopJobs»). Если healthz начнёт отвечать 503, docker или systemd перезапустят бот, и задачи молча включатся снова вопреки решению админа. Поэтому состояние планировщика отдаётся метрикой `sbot_scheduler_running`, а на healthz не влияет. Алерт на «0 дольше N минут» настраивается в Prometheus.

### D4. Точки записи
- `runlog.Recorder.Record`: `ReportRuns.WithLabelValues(kind, status).Inc()` и `ReportRunDuration.Observe(finished-started)` до записи в БД, чтобы метрика была и при ошибке БД.
- `collector.Collector.collect`: замер вокруг `fetchFn`, метка `string(crd.Type)`; ошибка → `CollectorErrors`.
- `retry.Do`: каждый повтор → `DeliveryRetries(op)`, итоговая ошибка (кроме отмены контекста) → `DeliveryFailures(op)`.
- `sheduler.Sheduler.Start` → `SchedulerRunning.Set(1)`, `Stop` → `Set(0)`.
- `internal/cli/run.go`: `BuildInfo.WithLabelValues(version, commit).Set(1)` после настройки логгера.

### D5. Конфиг
`internal/config/config.go`: `Metrics metrics.Config` (`yaml:"metrics"`): `Enabled bool` (`enabled`, env `METRICS_ENABLED`, по умолчанию `false`) и `Addr string` (`addr`, env `METRICS_ADDR`, `env-default:"127.0.0.1:9090"`).

Пересмотр при реализации (2026-10-24, решение пользователя): изначально сервер отключался пустым `addr`, но cleanenv подставляет `env-default` вместо пустой строки, так что выключить сервер было нельзя. Флаг `enabled` с дефолтом `true` не работает по той же причине (`false` — нулевое значение, затирается дефолтом; `*bool` cleanenv не поддерживает). Поэтому `enabled` по умолчанию `false`, как `max.enabled`: сервер включается явно. Образец в `config/config.example.yaml` с комментарием, что для доступа из другого контейнера нужен `0.0.0.0:9090` и закрытый порт.

### D6. Зависимость
`go get github.com/prometheus/client_golang@v1.12.1` (без смены версии, только прямая зависимость) и `go mod tidy`. Повышение версии в это изменение не входит.

## Risks / Trade-offs

- [По умолчанию `127.0.0.1` — из docker-сети недоступно] → безопасный дефолт. Рабочий адрес задаётся в конфиге, `deploy/` не трогаем.
- [Глобальные метрики в тестах накапливаются между тестами] → тесты сравнивают разницу «до/после», а не абсолютное значение.
- [client_golang 1.12.1 старый] → API `promauto`/`promhttp`/`testutil` стабилен. Обновление — отдельный бамп.

## Migration Plan

Без миграций БД. В рабочем конфиге при необходимости задать `metrics.addr`. Healthcheck в docker и systemd подключается отдельно, после этого изменения, по явной просьбе (`deploy/` вне объёма).
