# Design

## Context

- `models.Message` — данные (получатели, текст, файлы, картинки, meta) плюс методы отправки `Send/sendTg/sentMax/sendSMB/sendSMTP` и интерфейсы `TgSender/SmbSender/SMTPSender/MaxSender/senderProvider`.
- Оркестратор держит `models.SenderProvider` и использует его дважды: `msg.Send(ctx, o.snd)` в `deliver` и `o.snd.Tg().SendText` в `notifyTest`.
- `internal/delivery/smtp` не импортирует `models`, поэтому новый пакет `delivery` может импортировать и `models`, и подпакеты каналов без циклов.

## Goals / Non-Goals

**Goals:** `models` не знает о каналах доставки; изменение канала локализуется в `internal/delivery`.

**Non-Goals:** менять адаптеры каналов (`telegram`, `max`, `smb`, `smtp`); менять поведение отправки и формат письма.

## Decisions

1. **Пакет `internal/delivery`, тип `Dispatcher`** с полями tg/smb/smtp/max (интерфейсы отправителей переезжают сюда как есть) и `NewDispatcher(tg, smb, smtp, max)`. Метод `Send(ctx, *models.Message) ([]models.SentMessage, int, error)` — перенос `Message.Send` без изменения логики. Помощники `sendTg/sendMax/sendSMB/sendSMTP` становятся неэкспортируемыми функциями пакета. Альтернатива — интерфейс `Channel` на каждый тип получателя с реестром; отклонена как лишняя абстракция при четырёх фиксированных каналах.
2. **`Dispatcher.TgText(ctx, chatID int64, text string) error`** — для служебной строки пробного запуска; оркестратору не нужен весь `TgSender`.
3. **Оркестратор** объявляет у себя `type MessageSender interface { Send(ctx, *models.Message) (...); TgText(...) error }` и принимает его в `New`. Мок — `go tool mockery`, если пакет уже так мокается; иначе — фейк в тесте, как сейчас (`fakeTg`, `fakeSmb`).
4. **`models`** сохраняет `Message`, `NewMessage`, `Recipient`, `SentMessage` и другие данные; удаляются `send.go`, интерфейсы отправителей и методы отправки. Хелпер `deRef` переезжает туда, где используется.
5. **Тесты** `models/message_test.go` про отправку (`TestMessageSendDelivered`, `TestMessageSendMaxPartial`) переезжают в `internal/delivery/dispatch_test.go`; тесты конструктора/данных остаются в `models`.

## Risks / Trade-offs

- [Большой перенос кода — риск потерять ветку поведения] → перенос без правок логики, тесты переезжают вместе с кодом и должны остаться зелёными без изменения ожиданий.
