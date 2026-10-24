# Design

## Context

См. proposal.md. Источник для `http` — любой URL из БД, значит тело ответа и редиректы контролирует внешний сервер. `sameHostRedirect` сравнивает только `URL.Host`; `parseXML` строит дерево без ограничения глубины, а `xnode.value()` рекурсивен; xlsx открывается `excelize.OpenReader` без лимитов распаковки.

## Decisions

- D1. `sameHostRedirect`: `req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host` → `http.ErrUseLastResponse` (ответ 3xx даёт ошибку статуса, как уже для чужого хоста). Привязку секрета к схеме не добавляем: без редиректа схему карточки задаёт тот, кто пишет в БД, а секрет уходит на свой хост; это осознанный остаток (см. Risks).
- D2. `parseXML`: константа `maxXMLDepth = 1000`; при `len(stack) >= maxXMLDepth` на `StartElement` — ошибка `xml nesting deeper than 1000`. Рекурсия `value()`/`funcs.Flatten` тогда ограничена той же глубиной.
- D3. xlsx: `excelize.OpenReader(r, excelize.Options{UnzipSizeLimit: maxBody, UnzipXMLSizeLimit: maxBody})` — excelize сам возвращает ошибку при превышении.

## Risks / Trade-offs

- [Секрет может уйти по http, если карточку в БД сразу направили на `http://<хост секрета>`] → требует записи в БД и позиции в сети; принято. Привязку к схеме можно добавить позже вместе с хостом (`ctl secret set -host https://...`).
- [Легитимный XML глубже 1000 уровней перестанет разбираться] → таких отчётных API не встречали.
