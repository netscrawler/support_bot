# Tasks

> Проверка: `CGO_ENABLED=1 go build ./... && go test ./... && go tool golangci-lint run`. Ponytail full, тест вместе с кодом.

## 1. Защита от враждебного ответа

- [x] 1.1 `internal/collector/httpcoll/collector.go`: `sameHostRedirect` запрещает смену схемы (D1). Тест в `collector_test.go`: два `httptest` сервера на одном хосте недоступны для https→http, поэтому проверить сам `sameHostRedirect` юнит-тестом на `http.Request` (https→http, http→https, тот же origin, другой хост). Проверка: `go test ./internal/collector/httpcoll/...`.
- [x] 1.2 `internal/collector/httpcoll/decode.go`: лимит глубины XML 1000 (D2). Тест в `decode_test.go`: 100000 вложенных `<a>` → ошибка с `1000`, 1000 уровней — ок. Проверка: `go test ./internal/collector/httpcoll/...`.
- [x] 1.3 `internal/collector/httpcoll/decode.go`: лимиты распаковки xlsx = `maxBody` (D3). Тест: при заниженном `maxBody` обычный xlsx из теста даёт ошибку, при обычном — разбирается. Проверка: полная проверка выше, вывод приложить.

## Workflow follow-up

- `/opsx:archive`, коммит `fix(collector): защита http-сборщика от редиректа на http, глубокого XML и xlsx-бомбы`.
