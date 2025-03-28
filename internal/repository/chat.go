package repository

import (
	"context"
	"errors"
	"fmt"
	"support_bot/internal/database/postgres"
	"support_bot/internal/models"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type Chat struct {
	storage *postgres.Storage
	log     *zap.Logger
}

func NewChat(s *postgres.Storage, log *zap.Logger) Chat {
	return Chat{
		storage: s,
		log:     log,
	}
}

func (c *Chat) Create(ctx context.Context, chat *models.Chat) error {
	const op = "repository.Chat.Create"

	query, args, err := c.storage.Builder.
		Insert("chats").
		Columns("chat_id", "title", "type", "description", "is_active").
		Values(
			chat.ChatID,
			chat.Title,
			chat.Type,
			chat.Description,
			chat.IsActive,
		).
		ToSql()
	if err != nil {
		c.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))
		return err
	}

	_, err = c.storage.Db.Exec(ctx, query, args...)
	if err != nil {
		c.log.Error(fmt.Sprintf("%s error exec query: %s", op, err.Error()))
		return err
	}

	return nil
}

func (c *Chat) GetByTitle(ctx context.Context, title string) (*models.Chat, error) {
	const op = "repository.Chat.GetByTitle"
	query, args, err := c.storage.Builder.
		Select(
			"id",
			"chat_id",
			"title",
			"type",
			"description",
			"is_active",
		).
		From("chats").
		Where(squirrel.Eq{"title": title}).
		ToSql()
	if err != nil {
		c.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))

		return nil, models.ErrInternal
	}

	row := c.storage.Db.QueryRow(ctx, query, args...)

	var chat models.Chat
	if err := row.Scan(
		&chat.ID,
		&chat.ChatID,
		&chat.Title,
		&chat.Type,
		&chat.Description,
		&chat.IsActive,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.log.Debug(fmt.Sprintf("%s | Not found chat with title %s", op, title))
			return nil, models.ErrNotFound
		}
		return nil, models.ErrInternal
	}
	return &chat, nil
}

func (c *Chat) GetAll(ctx context.Context) ([]models.Chat, error) {
	const op = "repository.Chat.GetAll"
	query, args, err := c.storage.Builder.
		Select(
			"id",
			"chat_id",
			"title",
			"type",
			"description",
			"is_active",
		).
		From("chats").
		ToSql()
	if err != nil {
		c.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))

		return nil, models.ErrInternal
	}

	rows, err := c.storage.Db.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.log.Error(fmt.Sprintf("%s | %s", op, err))
			return nil, models.ErrNotFound
		}
		c.log.Error(op, zap.Error(err))
		return nil, models.ErrInternal
	}
	defer rows.Close()

	chats := make([]models.Chat, 0)
	for rows.Next() {
		var chat models.Chat
		if err := rows.Scan(
			&chat.ID,
			&chat.ChatID,
			&chat.Title,
			&chat.Type,
			&chat.Description,
			&chat.IsActive,
		); err != nil {
			c.log.Error(fmt.Sprintf("%s | %s", op, err.Error()))
			continue
		}
		chats = append(chats, chat)
	}
	c.log.Info(fmt.Sprintf("%s : successfully got %d chats", op, len(chats)))
	return chats, nil
}

func (c *Chat) Delete(ctx context.Context, chatID int64) error {
	const op = "repository.Chat.Delete"

	query, args, err := c.storage.Builder.
		Delete("chats").
		Where(squirrel.Eq{"chat_id": chatID}).
		ToSql()
	if err != nil {
		c.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))
		return err
	}

	_, err = c.storage.Db.Exec(ctx, query, args...)
	if err != nil {
		c.log.Error(fmt.Sprintf("%s error exec query: %s", op, err.Error()))
		return err
	}

	return nil
}
