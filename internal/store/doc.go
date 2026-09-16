// Package store содержит sqlc-ориентированный слой доступа к БД для отчётов —
// флагманский пакет фазы 1 миграции на sqlc.
//
// Он оборачивает сгенерированный *sqlcgen.Queries в узкий тип ReportStore,
// который предоставляет только те методы, что реально нужны потребителям
// (оркестратору, планировщику, HTTP API, Telegram-боту), и нигде за пределы
// пакета не выпускает сам sqlcgen.Queries. ReportStore заменяет собой три
// независимые ранее реализации загрузки отчётов (internal/repository,
// internal/orchestrator/orchestrator_repository.go,
// internal/tg_bot/repository/report_repository.go) единой точкой сборки
// модели models.Report из строк, читаемых через sqlc.
//
// Основные публичные типы:
//   - ReportStore — хранилище отчётов поверх *pgxpool.Pool.
//
// Основные публичные функции/методы:
//   - NewReportStore — конструктор для продакшена (fx-DI).
//   - ReportStore.Load / LoadActive / GetByID / GetByName / GetByPublicID /
//     LoadByEvent / LoadPaged / GetLinkedCrons — чтение отчётов и их
//     зависимостей (queries, recipients, exports, crons, pipeline).
//   - ReportStore.Create — создание отчёта и всех его зависимостей одной
//     транзакцией с get-or-create резолвингом по естественным ключам.
//   - ExecTx — обёртка над pgx-транзакцией, разделяемая всеми Store; заменяет
//     internal/pkg/uow.
//
// Место в архитектуре: store — часть пайплайна отчётов (Report Store),
// используется на всех этапах, где нужен доступ к таблице reports и связанным
// с ней таблицам (queries, exports, crons, recipients, pipelines). Ошибка
// pgx.ErrNoRows транслируется в models.ErrNotFound на границе пакета
// (translateNoRows) — вызывающий код за пределами store никогда не проверяет
// pgx.ErrNoRows напрямую.
package store
