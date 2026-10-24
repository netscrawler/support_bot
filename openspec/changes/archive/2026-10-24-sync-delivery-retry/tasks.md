# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной.
> Сгенерированный код не править, моки — через `go tool mockery`. Ponytail full.

## 1. retry.Do

- [x] 1.1 `internal/pkg/retry/retry.go`: `Delays`, `Do`, `Permanent`, `After` по D1. Тесты писать до кода или вместе с ним: успех со второй попытки; все попытки упали → последняя ошибка и `len(delays)+1` вызовов; `Permanent` → один вызов и ошибка без обёртки; `After(d)` ждёт d, а не delays[i]; отмена контекста в паузе → сразу `ctx.Err()`; пустой `Delays` → один вызов. Очередь пока не удалять. Проверка: `go test ./internal/pkg/retry/...`.

## 2. Каналы

- [x] 2.1 Telegram: `classify.go` (D2) и `retry.Do` во всех методах `ChatAdaptor`, включая `DeleteMsg`, по D3. Конструктор принимает `retry.Delays`. Тесты классификации: 400/403 → permanent, 429 `retry_after=3` → After 3s, 429 `retry_after=120` → permanent, сеть → повторяемая. Проверка: `go test ./internal/delivery/telegram/...`.
- [x] 2.2 Max: `retry.Do` во всех методах, включая `DeleteMsg`. Конструктор принимает `retry.Delays`. Проверка: `go build ./...`, `go test ./internal/delivery/max/...`.
- [x] 2.3 SMTP и SMB: `retry.Do` в `Send`/`Upload`, классификация SMTP 5xx (D2). Тест классификации SMTP: `textproto.Error{Code: 550}` → permanent, `421` → повторяемая. Проверка: `go test ./internal/delivery/...`.

## 3. Конфиг и сборка

- [x] 3.1 `internal/config/config.go`: `Delivery.Retry.Delays` по D4, тест разбора YAML и значения по умолчанию в `config_test.go`. `internal/app/module_delivery.go`: передать `cfg.Delivery.Retry.Delays` в четыре адаптера. Удалить `newRetry` и очередь из `internal/pkg/retry/retry.go` (D5). `config/config.example.yaml`: секция `delivery.retry`. Проверка: `go build ./...`, `go test ./internal/config/... ./internal/pkg/retry/...`, `grep -rn "retry\.New(\|retry\.NewTask\|AddRateLimit" internal` пусто.

## 4. Интеграция

- [x] 4.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Ручной прогон `task run`: отчёт в Telegram при кратком отключении сети (например, `iptables`/отключение Wi‑Fi на 3s) доходит, в логе warn о повторе; отчёт в чат, где бота нет, падает сразу (403) без повторов. Результат записать сюда.
  - 2026-10-24 авто: `go build ./...` OK; `go test ./...` красный только в базовой линии (`internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`); `golangci-lint --new-from-rev=HEAD` 0 issues; grep `retry.New(|retry.NewTask|AddRateLimit` пуст.
  - Ручной прогон: пропущен по решению пользователя 2026-10-24.

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(delivery): синхронный повтор отправки во всех каналах`.
