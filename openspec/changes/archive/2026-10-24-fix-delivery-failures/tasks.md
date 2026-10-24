# Tasks

## 1. MAX: частичная доставка вложений

- [x] 1.1 Написать `internal/delivery/max/max_test.go` (`httptest.Server` + `maxcli.NewApi(..., WithBaseURL)`): один файл из двух не загрузился → сообщение с одним вложением ушло, ошибка содержит имя файла; все не загрузились → `Messages.Send` не вызывался; документ прикреплён с типом `file`; последняя ошибка `Send` возвращается после повторов; отмена ctx прерывает повторы. Убедиться, что тесты падают на текущем коде (первопричина подтверждена).
- [x] 1.2 Исправить `sendUploads` (`errors.Join` ошибок загрузки, пропуск отправки без токенов, тип вложения по `kind`) — проверка: тесты 1.1 зелёные.
- [x] 1.3 `internal/models/message.go`: `sentMax` объединяет ошибки и возвращает частично отправленные сообщения; ветка `MaxRecipient` в `Send` сохраняет их в `sentMsg` при ошибке, `delivered` не растёт. Тест в `internal/models` (фейковый `MaxSender`): ошибка + сообщение → `sentMsg` содержит его, `delivered == 0`, `sendErr != nil`.

## 2. SMB: восстановление соединения и запись

- [x] 2.1 Написать `internal/delivery/smb/client_test.go` с фейковым `share`: `cleanup()` не отменяет ctx монитора, а `Close()` отменяет; `Upload` повторяет запись после ошибки и проходит; отмена ctx прерывает повторы; `Upload` при `fs == nil` возвращает ошибку без паники. Убедиться, что тест на `cleanup` падает на текущем коде.
- [x] 2.2 Ввести интерфейс `share{WriteFile, Stat, Umount}`, перевести `write` на `WriteFile`, добавить проверку `fs == nil`, убрать `smb.cancel()` из `cleanup()`. Над полем `fs` оставить комментарий `shortcut:` про гонку с монитором. Проверка: тесты 2.1 зелёные.

## 3. Проверка

- [x] 3.1 `CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./... && go tool golangci-lint run` зелёные; у `internal/delivery/max` и `internal/delivery/smb` нет `[no test files]`. (падения `internal/evaluator` TestEvaluator_Evaluate/none_data и `internal/pkg/text` TestExecuteTemplate/date_month есть и на чистом HEAD, вне этого изменения)
- [x] 3.2 `code-simplifier` на изменённых файлах, затем повторить 3.1.

## Workflow follow-up

- `/opsx:archive`, затем коммит `fix(delivery): …` на русском.
