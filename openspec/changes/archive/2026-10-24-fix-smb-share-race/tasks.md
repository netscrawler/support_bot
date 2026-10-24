# Tasks

## 1. Мьютекс соединения SMB

- [x] 1.1 Тест в `internal/delivery/smb/client_test.go`: параллельно `Upload` и `cleanup()`/присваивание `fs` (через фейковую шару) под `go test -race` — подтвердить, что `-race` ловит гонку на текущем коде.
- [x] 1.2 Добавить `sync.RWMutex`, `RLock` в `write` и в проверке монитора, `Lock` в `cleanup` и при присваивании полей в `connect`; убрать `shortcut:`. Проверка: `CGO_ENABLED=1 go test -race ./internal/delivery/smb/` зелёный.

## 2. Проверка

- [x] 2.1 `CGO_ENABLED=1 go build ./... && CGO_ENABLED=1 go test ./... && go tool golangci-lint run` зелёные.
- [x] 2.2 Упрощение изменённых файлов (дифф из нескольких строк — проверен вручную), затем повторить 2.1.

## Workflow follow-up

- `/opsx:archive`, коммит `fix(smb): …`.
