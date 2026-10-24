# Design

## Context

- `internal/pkg/retry.Retry` — очередь задач с воркерами, backoff и rate limit. Создаётся в `app/module_collectors.go:newRetry` и передаётся только в `telegram.NewChatAdaptor` и `maxadp.New`. `Retry.Start` нигде не вызывается, поэтому воркеров нет и `Submit` просто наполняет буфер на 100 задач. Плюс `AddRateLimit` у Telegram и Max перезаписывает общий лимит.
- В `telegram.ChatAdaptor` каждый метод (`SendText`, `SendRichText`, `SendMedia`, `SendDocument`, `DeleteMsg`) содержит копию вызова API внутри `Submit`. В `max.Adaptor` так же.
- `smtp.Sender.Send` и `smb.SMB.Upload` повторов не имеют. У SMB есть отдельный монитор переподключения (`reconn.go`).
- Ошибка Telegram — `*telegoapi.Error{ErrorCode, Parameters.RetryAfter}` (telego v1.11). SMTP — `net/smtp` + `*textproto.Error{Code}`.
- `retry.RoundTripper` используется коллекторами (metabase и др.) и к этому изменению не относится.

Решения: grilling 2026-10-23, Q5, Q16, Q26.

## Goals / Non-Goals

**Goals:** один простой синхронный повтор, который используют все каналы; удалить неработающую очередь.

**Non-Goals:** повторы в коллекторах, сохранение неотправленного в БД для поздней доставки, идемпотентность.

## Decisions

### D1. `retry.Do` вместо очереди
В `internal/pkg/retry/retry.go` очередь (`Retry`, `Task`, `Config`, `Policy*`, `Backoff*`, `ErrQueueFull`) заменяется на:
```go
type Delays []time.Duration // паузы между попытками; попыток = len+1

// Do вызывает fn, пока она не вернёт nil, Permanent-ошибку или не кончатся паузы.
func Do(ctx context.Context, delays Delays, log *slog.Logger, op string, fn func(context.Context) error) error

// Permanent помечает ошибку как неповторяемую; After — как повторяемую через d.
func Permanent(err error) error
func After(err error, d time.Duration) error
```
Классификацию делает канал: он оборачивает ошибку в `Permanent`/`After` внутри `fn`. Так `retry` не зависит от telego и smtp. Ошибки контекста (`ctx.Err() != nil`) `Do` считает неповторяемыми сам. Пауза ждёт через `select` на `ctx.Done()`. Каждый повтор пишет warn с `op` (например, `telegram.SendDocument`), номером попытки и ошибкой. Наружу возвращается исходная ошибка, без обёртки `Permanent`. `round_tripper.go` не трогать. Альтернатива — декораторы над интерфейсами `TgSender`/`MaxSender`/… — отвергнута: 9 обёрток, а внутри адаптеров повтор заменяет уже существующие копии вызовов, так что кода становится меньше. (агент)

### D2. Классификация по каналам
- Telegram (`internal/delivery/telegram/classify.go`): `errors.As(err, *telegoapi.Error)`: коды 400, 401, 403, 404 → `Permanent`; 429 с `RetryAfter ≤ 60` → `After(RetryAfter s)`, больше 60 → `Permanent`; остальное повторяемое.
- Max: всё повторяемое, кроме ошибок контекста (формат ошибок клиента Max не типизирован). (агент)
- SMTP: `*textproto.Error` с `Code >= 500` → `Permanent`.
- SMB: всё повторяемое. Если монитор переподключения ещё не поднял сессию, следующая попытка через паузу получит новую.

### D3. Применение в адаптерах
- Telegram и Max: каждый вызов API заворачивается в `retry.Do`. Rate limiter (`ca.rl.Wait`) вызывается внутри `fn`, на каждую попытку. Блоки `Submit` с копиями параметров удаляются. В `SendDocument` повтор оборачивает отправку одного файла в цикле, поэтому уже ушедшие файлы не дублируются. `SendMedia` повторяет всю группу.
- `DeleteMsg` в обоих адаптерах тоже через `retry.Do`.
- SMTP: `Send` заворачивает соединение и отправку одного письма. SMB: `Upload` заворачивает запись файла.
- Конструкторы получают `retry.Delays` вместо `*retry.Retry`.

### D4. Конфиг (Q16)
`internal/config/config.go`: `Delivery struct{ Retry struct{ Delays []time.Duration } }` с yaml `delivery.retry.delays`, по умолчанию `[1s, 5s, 15s]`. Пустой список означает «без повторов». Если cleanenv не разбирает `[]time.Duration` из `env-default`, значение по умолчанию ставится в коде (`if cfg.Delivery.Retry.Delays == nil`), а `Delays` в env не поддерживается. Это проверяется тестом в `config_test.go`. (агент)

### D5. Удаление
`newRetry` из `module_collectors.go` и параметр `retr *retry.Retry` из провайдеров доставки удаляются. `retr.AddRateLimit` больше не нужен: каждый адаптер хранит свой `rl`.

## Risks / Trade-offs

- [Дубль при таймауте после фактической доставки] → принято (Q16).
- [Отчёт держит слот оркестратора до ~21s плюс время попыток] → при 8 слотах и недоступном Telegram очередь событий тормозит. Это лучше, чем терять отчёты. Паузы настраиваются.
- [Max: повтор и на «окончательных» ошибках] → максимум 3 лишние попытки за 21s.
- [Удаляется публичный API пакета `retry`] → внешних пользователей нет. `grep -rn "retry\.\(New\|Task\|Policy\)"` после правки должен быть пустым.

## Migration Plan

Без миграций. Необязательно: `delivery.retry.delays` в рабочем конфиге.
