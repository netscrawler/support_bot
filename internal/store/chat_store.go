package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChatStore реализует service.ChatProvider поверх sqlc-запросов chat.sql.
// Заменяет internal/tg_bot/repository.ChatRepository, чья GetByTitle
// содержала баг: nil-указатель передавался в sqlx.GetContext и приводил к
// панике при любом совпадении. Генерируемый sqlc-метод возвращает строку
// значением, поэтому такой класс ошибок здесь структурно невозможен.
type ChatStore struct {
	q   *sqlcgen.Queries
	log *slog.Logger
}

func NewChatStore(pool *pgxpool.Pool, log *slog.Logger) *ChatStore {
	return &ChatStore{
		q:   sqlcgen.New(pool),
		log: log.With(slog.Any("module", "store.chat")),
	}
}

func (s *ChatStore) Create(ctx context.Context, chat *models.TgChatDTO) error {
	_, err := s.q.CreateChat(ctx, sqlcgen.CreateChatParams{
		ChatID:      chat.ChatID,
		Title:       &chat.Title,
		Type:        chat.Type,
		Description: chat.Description,
		IsActive:    chat.IsActive,
		ChType:      chat.ChType,
	})
	if err != nil {
		return fmt.Errorf("create chat: %w", err)
	}

	return nil
}

// Exists проверяет наличие чата по chat_id — реальному уникальному
// естественному ключу таблицы chats (в отличие от title, который может
// повторяться у разных чатов). Используется service.Chat.Add при
// регистрации, чтобы не путать разные Telegram-группы с одинаковым
// отображаемым названием. Переиспользует тот же sqlc-запрос, что и
// ReportStore.getOrCreateChat.
func (s *ChatStore) Exists(ctx context.Context, chatID int64) (bool, error) {
	_, err := s.q.FindChatIDByChatID(ctx, chatID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("find chat by id: %w", err)
	}

	return true, nil
}

func (s *ChatStore) GetByTitle(ctx context.Context, title string) (*models.TgChatDTO, error) {
	row, err := s.q.GetChatByTitle(ctx, &title)
	if err != nil {
		return nil, translateNoRows(err)
	}

	chat := mapChatRow(
		row.ID, row.ChatID, row.Title, row.Type, row.Description, row.IsActive, row.ChType,
	)

	return &chat, nil
}

// GetAll возвращает чаты с is_active = false — это осознанное, зафиксированное
// на этапе планирования архитектуры поведение (не баг): неиспользуемый
// вариант с is_active = true в старом коде был мёртвым кодом и в sqlc-версию
// не переносится.
func (s *ChatStore) GetAll(ctx context.Context) ([]models.TgChatDTO, error) {
	rows, err := s.q.ListChats(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}

	chats := make([]models.TgChatDTO, 0, len(rows))
	for _, row := range rows {
		chats = append(chats, mapChatRow(
			row.ID, row.ChatID, row.Title, row.Type, row.Description, row.IsActive, row.ChType,
		))
	}

	return chats, nil
}

func (s *ChatStore) Delete(ctx context.Context, chatID int64) error {
	if err := s.q.DeleteChat(ctx, chatID); err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}

	return nil
}

func mapChatRow(
	id int32, chatID int64, title *string, chatType string,
	description *string, isActive bool, chType string,
) models.TgChatDTO {
	return models.TgChatDTO{
		ID:          int(id),
		ChatID:      chatID,
		Title:       derefStr(title),
		Type:        chatType,
		Description: description,
		IsActive:    isActive,
		ChType:      chType,
	}
}
