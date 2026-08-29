# План развития: единый layout-экспорт (xlsx / pdf / html из одного описания)

> Документ-план для агентов и разработчиков. Обновляйте статус чекбоксов по мере выполнения работ.
> Спецификация grid-движка для HTML/PDF: `internal/exporter/html/task.md`.

## Цель

Автор описывает отчёт декларативным JSON-layout (блоки: таблица, график, метрика, текст с позицией на сетке 12 колонок). Система сама рендерит его в **xlsx**, **html** и **pdf** (pdf — через html → chromium). Старые `Order` / `Template` конвертируются в layout автоматически при загрузке и удаляются из модели — остаётся один кодовый путь.

Принятые решения (подтверждены владельцем):

1. **Формат layout**: декларативный JSON + каноническая блочная модель в Go. HTML-шаблоны с `reportGrid` остаются escape-hatch'ем и строят те же блоки.
2. **Совместимость**: чистый переход — автоконвертация легаси (`Order`, inline-шаблоны) в layout при загрузке, затем выпиливание легаси-полей.
3. **PDF-движок**: только chromium (`-tags chromium`); wkhtmltopdf-билдтег остаётся как fallback.
4. **Блоки v1**: `table`, `chart`, `metric`, `text` (+ `raw_html` как escape-hatch). `image` — в v2.

---

## Фаза 0 — стабилизация конвейера ✅ (завершена)

- [x] **0.1 Подключить экспортер**: `exporter.NewEngine(cfg.ChromePath, nil)` собирается в `internal/app/app.go`; `generator.New` принимает `Exporter` и сохраняет в поле (раньше поле не заполнялось → nil-паника на первом отчёте).
- [x] **0.2 Убрать глобальное состояние**: `defaults` в `internal/exporter/exporter.go` стал локальной переменной внутри `NewEngine`; стек коллекторов `var collectors` в html-пакете заменён на per-call `gridState` (`layout_funcs.go`), т.к. генератор запускает 4 конкурентных воркера.
- [x] **0.3 Починить text-экспортер**: парсит `Template.TemplateText` (была регрессия — парсился `format.Format`); тип данных берётся из `Template.Type`.
- [x] **0.4 PDF DI**: `pdf.New(path, html)` использует переданный путь (chromium-билд больше не читает глобальный конфиг; `GetChromePathFromConfig` удалён); wkhtmltopdf-билд вызывает `wkhtmltopdf.SetPath`; добавлен `chromedp.NoSandbox` для контейнеров; убран проглатывающий панику `defer recover` в `buildChartBlock`.
- [x] **0.5 Dockerfile → chromium**: оба Dockerfile собирают с `-tags chromium`, рантайм ставит `chromium` вместо `wkhtmltopdf`.
- [x] **0.6 Тесты/мусор**: `exporter_test.go` включён (энд-ту-энд по `example.gohtml` + пагинация таблиц), `example_out.html` удалён и добавлен в `.gitignore`, пустой `func.go` удалён.
- [x] **0.7 Golden-тесты**: xlsx (листы, типизация, excelize-таблица), Engine (диспетчеризация форматов, override кастомными экспортерами, неизвестный формат).
- [x] **0.8 Починен завязанный на дату тест** `parse_func_test.go` (`date_month`).
- [x] **0.9 `.golangci.yml` переведён в валидный v2-формат** (v1-синтаксис молча игнорировался: `linters-settings` → `linters.settings`, `disable-all` → `default: none`).
- [x] **0.10 API `table` приведён к данным пайплайна**: `table(rows []map[string]any, columns []string, options)` с колонками `"field:Заголовок"` — старая сигнатура `([]string, [][]string)` не могла быть вызвана из шаблона, т.к. `models.Dataset` хранит только `[]map[string]any`. Pie-график теперь явно отклоняет мульти-серии (раньше проходило только благодаря проглатыванию паники).

## Фаза 1 — каноническая модель layout (ядро)

- [x] **1.1 `internal/models/layout.go`** (заменить пустой `export_layout_template.go`):
  - `Layout{Version int, Page PageConfig, Blocks []Block}`;
  - `PageConfig{Format:"A4", Orientation, PaddingMM, Header/Footer}`;
  - `Block{ID, Type, Dataset (ключ из models.Dataset), Pos GridPosition{X,Y,W,H}}` + типизированные опции:
    - `TableBlock{Columns []Column{Field, Title, Width, Format, Align}, Sort, Limit}` — splittable;
    - `ChartBlock{Kind: line|bar|pie, XField, Series []Series{Field,Title,Color}, Stacked}`;
    - `MetricBlock{Field, Label, Format}`;
    - `TextBlock{Text (go-template)}`;
    - `RawHTMLBlock` — html/pdf рендерят как есть, xlsx — как текст.
- [x] **1.2 Реестр блоков** (пакет `internal/layout`): `map[BlockType]Definition{Validate}`; `Layout.Validate(datasetKeys)` — известные типы, границы сетки (0..11 / w+h ≤ 12), уникальные ID, существование датасетов.
- [x] **1.3 Автоконвертация легаси**: `LayoutFromOrder(Order)` (таблица на весь лист, лист = датасет), `LayoutFromTemplate(*Template)` (text/rich_text → TextBlock). Табличные тесты обязательны.
- [x] **1.4 Пример** `internal/models/layout.json` — реальный документ; обновить `ctl create` / `models/report_example.go`.

## Фаза 2 — рендереры поверх модели

- [x] **2.1 HTML**: `Layout → []Block → grid-движок (html/layout.go) → Pages → HTML`. В `styles.go` добавлено правило `.report-page` + `break-after: page` и `@page` из `PageConfig`. Chart.js уже есть (`chart.go`).
- [ ] **2.2 PDF**: без изменений поверх html (chromedp); layout рендерится одним файлом.
- [ ] **2.3 XLSX** (основная новая работа): table → лист + excelize-таблица со стилями; chart → `excelize.AddChart` (заполнить пустой `charts.go`); metric → KPI-ячейки; text/raw_html → текст. Удалить или реализовать заглушку `xlsx/layout_definition.go`.
- [ ] **2.4 text/png/csv поверх layout**: text — TextBlock; png — таблицы через существующий `generateTableImageFromMatrix`; csv — таблицы → файлы.

## Фаза 3 — хранение, модель отчёта, DSL

- [ ] **3.1 Миграция БД**: таблица `layouts(id, title, layout jsonb, version int)`; `reports_export.layout_id` (nullable) — чинит веерную привязку шаблонов (сейчас `report_templates` джойнит каждый шаблон отчёта к каждому его экспорту, см. `repository/report.go` `loadExports`).
- [ ] **3.2 `models.Export`**: добавить `Layout *Layout`; при загрузке из БД конвертировать легаси Order/Template → Layout; затем выпилить `Template`/`Order` из модели, репозиториев и DSL.
- [ ] **3.3 repository + service/report_manager**: CRUD layouts, persist при `ctl apply`.
- [ ] **3.4 CLI ctl**: export/apply переносят layout в JSON-дампе; `validate` — реальная валидация layout через реестр блоков (сейчас заглушки в `service/report_validator.go`) + белый список форматов.

## Фаза 4 — инструменты и качество

- [ ] **4.1 live-server**: превью layout JSON (`/preview/html|pdf|xlsx`) на живых данных коллектора; hot-reload через существующий fsnotify.
- [ ] **4.2 Тесты**: golden-тесты рендереров по fixture'ам (layout + Dataset → эталон); юнит-тесты grid-движка (CanPlace/Place/Split/пагинация).
- [ ] **4.3 Документация**: раздел README «Layout-экспорт», обновить `report_example.go`, AGENTS.md.

## Фаза 5 — бэклог (после v1)

Image-блок, темы через токены `styles.go`, колонтитулы/нумерация страниц (task.md §21), версионирование схем layout, кэш датасетов между форматами.

---

## Ключевые факты аудита (не переоткрывать)

- Контракт экспортера: `Export(models.Dataset, models.Export) ([]models.Data, error)` (`internal/exporter/exporter.go`). `Dataset = map[string][]map[string]any` (ключ = title запроса). `Data` классифицируется по `sendKind` (text/rich_text/image/file) → доставка (TG: фото-альбом vs документы; SMTP/SMB — всё).
- PDF всегда идёт через html-экспортер (все три билдтега). Сборка: `task build` (CGO+musl, `-tags chromium`); тесты: `go test ./...`; линтер: `golangci-lint run` (конфиг в v2-формате, gofumpt/golines/funcorder — строгие).
- `_`-префиксные ключи датасета не попадают в экспорт; `_meta` используется для темы письма.
- `internal/sheduler` — опечатка в названии пакета сохраняется намеренно.
- Форматирование: `golangci-lint fmt ./...` перед коммитом.

## Правила для агентов

1. Перед правками html-экспортера читать `internal/exporter/html/task.md` (спецификация grid-движка).
2. Каждая фаза — отдельный коммит; после каждой фазы `go build -tags chromium ./... && go test ./... && golangci-lint run` должны быть зелёными.
3. Не вводить package-level mutable-состояние в exporter-пакетах (4 конкурентных воркера генератора).
4. Новые блоки регистрировать в реестре (Фаза 1.2), не хардкодить switch'и в рендерерах.
5. Держать сигнатуры template-функций совместимыми с `models.Dataset` (никаких `[]string`-аргументов из данных).
