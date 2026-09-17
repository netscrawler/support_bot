// Package store содержит sqlc-ориентированный слой доступа к БД для отчётов,
// чатов и пользователей — флагманский пакет фазы 1-2 миграции на sqlc.
//
// Он оборачивает сгенерированный *sqlcgen.Queries в узкие типы ReportStore,
// ChatStore и UserStore, которые предоставляют только те методы, что реально
// нужны потребителям (оркестратору, планировщику, HTTP API, Telegram-боту),
// и нигде за пределы пакета не выпускают сам sqlcgen.Queries. ReportStore
// заменяет собой три независимые ранее реализации загрузки отчётов
// (internal/repository, internal/orchestrator/orchestrator_repository.go,
// internal/tg_bot/repository/report_repository.go) единой точкой сборки
// модели models.Report из строк, читаемых через sqlc. ChatStore и UserStore
// аналогично заменяют internal/tg_bot/repository.ChatRepository/
// UserRepository, устраняя их зависимость от неинициализированного
// *sqlx.DB.
//
// Основные публичные типы:
//   - ReportStore — хранилище отчётов поверх *pgxpool.Pool.
//   - ChatStore — хранилище чатов Telegram-уведомлений (таблица chats).
//   - UserStore — хранилище пользователей бота (таблица users).
//   - SentMsgStore — хранилище отправленных сообщений (таблица sent_messages)
//     для оркестратора и удаления сообщений в конце дня.
//   - ScriptStore — хранилище Lua-скриптов (таблица lua_scripts): реализует
//     lua.PluginProvider для чтения и используется CLI-командой script save
//     для сохранения без перезаписи существующих скриптов.
//
// Основные публичные функции/методы:
//   - NewReportStore / NewChatStore / NewUserStore — конструкторы для
//     продакшена (fx-DI).
//   - ReportStore.Load / LoadActive / GetByID / GetByName / GetByPublicID /
//     LoadByEvent / LoadPaged / GetLinkedCrons — чтение отчётов и их
//     зависимостей (queries, recipients, exports, crons, pipeline).
//   - ReportStore.Create — создание отчёта и всех его зависимостей одной
//     транзакцией с get-or-create резолвингом по естественным ключам.
//   - ChatStore.Create / GetByTitle / GetAll / Delete — CRUD для чатов.
//   - UserStore.Create / Update / GetByUsername / GetByTgID / GetAll /
//     GetAllAdmins / Delete — CRUD для пользователей.
//   - SentMsgStore.SaveTgMsg / WithLockedMsgsToDelete / RemoveDeletedMessages /
//     MarkEndOfDayMsgDeleted — сохранение и удаление отправленных сообщений.
//   - ExecTx — обёртка над pgx-транзакцией, разделяемая всеми Store; заменяет
//     internal/pkg/uow.
//
// Место в архитектуре: store — часть пайплайна отчётов и Telegram-бота,
// используется на всех этапах, где нужен доступ к таблицам reports, chats,
// users и связанным с ними таблицам (queries, exports, crons, recipients,
// pipelines). Ошибка pgx.ErrNoRows транслируется в models.ErrNotFound на
// границе пакета (translateNoRows) — вызывающий код за пределами store
// никогда не проверяет pgx.ErrNoRows напрямую.
package store
