# Tasks

> Проверка после каждой задачи: `CGO_ENABLED=1 go build ./... && go test <затронутые пакеты> && go tool golangci-lint run --new-from-rev=HEAD`.
> Базовая линия (2026-10-23): `go test ./...` красный только в `internal/evaluator`, `internal/pkg/text`, `internal/processor/duck`. Если провал из базовой линии, задача не считается проваленной. Для `internal/evaluator` новые тесты должны быть зелёными, даже если старые красные.
> Предусловие: `improve-admin-test-run` в архиве. Сгенерированный код не править: `internal/db/sqlcgen/` — через `task generate`, моки — через `go tool mockery`. Ponytail full.

## 1. Период и CEL

- [x] 1.1 `internal/models/period.go` (D1). Табличные тесты `Range` на фиксированном `now`: каждый период; `month` 15 января; `week` и `yesterday` 1 марта високосного года. `ParsePeriod("")` → today, неизвестное → ошибка. Проверка: `go test ./internal/models/...`.
- [x] 1.2 `evaluator`: переменные и новая сигнатура `EvalStr` (D2), обновить `generator.Evaluator`, `models.Evaluator`, `Card.ResolveParams`, моки. Тесты: `=period_from` даёт ту же строку, что `=today()` для периода today; `=addDays(period_to, -1)` работает; значение без `=` возвращается как есть. Проверка: `go test ./internal/evaluator/... ./internal/models/... ./internal/generator/...` (новые тесты зелёные).

## 2. Хранение и проброс

- [x] 2.1 `migrations/init.sql/010_report_default_period.sql`, `sqlc.yaml`, колонка в select и `CreateReport`, `task generate`, маппинг в store, валидация и `report_example.go` (D4). Тесты: store на `pgxmock` читает и пишет `default_period`; валидатор отклоняет `weak`. Проверка: `go test ./internal/store/... ./internal/service/...`.
- [x] 2.2 `Report.Period`, `UsesPeriod`, период в `SpecialEventForLK`, `ProduceSpecialEvent`/`ProduceTestEvent`, выбор периода в генераторе и оркестраторе (D3). Тесты: генератор с `Period=""` и `DefaultPeriod=yesterday` передаёт вчерашний диапазон; оркестратор для `lk` берёт период из события, для `cron` — нет. Проверка: `go test ./internal/generator/... ./internal/orchestrator/...`.

## 3. Бот

- [x] 3.1 Клавиатура периода для админа и пользователя, разбор 5-го поля, проверка 64 байт, текст с периодом (D5). Тесты: разбор callback с кодом и без; отчёт без `UsesPeriod` → без клавиатуры; слишком длинное имя → запуск за `DefaultPeriod`. Проверка: `go test ./internal/tg_bot/...`.
- [x] 3.2 `README.md`, раздел про параметры запросов: `period_from`/`period_to`, четыре периода с точными границами, `default_period`, пример params для pg. Проверка: пример JSON проходит `sbot ctl validate`.

## 4. Интеграция

- [x] 4.1 Полная проверка `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run --new-from-rev=HEAD`, вывод приложить. Ручной прогон `task run` на БД с 007–010: pg-отчёт с `where created_at >= '${from}' and created_at < '${to}'` и `params {"from": "=period_from", "to": "=period_to"}`: в ЛК показан выбор периода, «Вчера» и «Месяц» дают разные данные; cron с `default_period: yesterday` берёт вчера; отчёт без period запускается сразу. Результат записать сюда.
  - Результат 2026-10-24: `go build ./...` ок; `go test ./...` красный только в базовой линии (`evaluator` — старый `TestEvaluator_Evaluate`, новый `TestEngine_EvalStr_Period` зелёный; `pkg/text`; `processor/duck`); линт `0 issues`. Пример из README проходит `ctl validate`, `default_period: "weak"` отклоняется.
  - На локальную `bottst` накатана `010_report_default_period.sql`, существующие отчёты получили `today`.
  - Ручной прогон в Telegram (выбор периода в ЛК и «📤 Получить», «Вчера»/«Месяц» дают разные данные, cron с `default_period: yesterday`, отчёт без period запускается сразу) — по решению пользователя выполняет пользователь. Сценарии покрыты юнит-тестами: `models` (границы периодов), `generator` (выбор периода), `orchestrator` (период события для lk, не для cron), `tg_bot/handlers` (`periodStep`, разбор callback, лимит 64 байта), `tg_bot/service` (период доходит до события).

## Workflow follow-up

- `code-simplifier` на изменённых файлах, затем повторная проверка.
- `/opsx:archive`, затем коммит `feat(report): выбор периода отчёта и период по умолчанию`.
- При деплое: накатить вручную `010_report_default_period.sql`.
