package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	models "support_bot/internal/models/notify"
)

type ChatRepository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewChatRepository(db *sqlx.DB, log *slog.Logger) *ChatRepository {
	l := log.With(slog.Any("module", "tg_bot.repository.chat"))

	return &ChatRepository{db: db, log: l}
}

func (c *ChatRepository) Create(ctx context.Context, chat *models.Chat) error {
	const query = `INSERT INTO chats (chat_id, title, type, description, is_active) 
	VALUES ( $1,$2,$3,$4,$5 );`

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("chat repository create: %w", err)
	}

	_, err := c.db.ExecContext(
		ctx,
		query,
		chat.ChatID,
		chat.Title,
		chat.Type,
		chat.Description,
		chat.IsActive,
	)
	if err != nil {
		return fmt.Errorf("creating chat : %w", err)
	}

	return nil
}

func (c *ChatRepository) GetByTitle(ctx context.Context, title string) (*models.Chat, error) {
	const query = `SELECT * FROM chats
WHERE title =$1
lIMIT 1;`

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("chat repository get by title: %w", err)
	}

	var chat *models.Chat

	err := c.db.GetContext(ctx, chat, query, title)
	if err != nil {
		return nil, fmt.Errorf("get by title : %w", err)
	}

	return chat, nil
}

func (c *ChatRepository) GetAll(ctx context.Context) ([]models.Chat, error) {
	const query = "SELECT * FROM chats;"

	err := ctx.Err()
	if err != nil {
		return nil, fmt.Errorf("chat repository get all: %w", err)
	}

	var chats []models.Chat

	err = c.db.SelectContext(ctx, &chats, query)
	if err != nil {
		return nil, fmt.Errorf("get all: %w", err)
	}

	return chats, nil
}

func (c *ChatRepository) GetAllActive(ctx context.Context) ([]models.Chat, error) {
	const query = "SELECT * FROM chats where is_active=true;"

	err := ctx.Err()
	if err != nil {
		return nil, fmt.Errorf("chat repository get all active: %w", err)
	}

	var chats []models.Chat

	err = c.db.SelectContext(ctx, &chats, query)
	if err != nil {
		return nil, fmt.Errorf("get all active: %w", err)
	}

	return chats, nil
}

func (c *ChatRepository) Delete(ctx context.Context, chatID int64) error {
	const query = "DELETE FROM chats WHERE chat_id = $1;"

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("chat repository delete: %w", err)
	}

	res, err := c.db.ExecContext(ctx, query, chatID)
	if err != nil {
		return fmt.Errorf("delete : %w", err)
	}

	if count, err := res.RowsAffected(); err == nil {
		c.log.DebugContext(ctx, "deleted chats", slog.Any("count", count))
	}

	return nil
}
