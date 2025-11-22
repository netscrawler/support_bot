package pgrepo

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	gen "support_bot/internal/infra/out/pg/gen"
	"support_bot/internal/models"
)

type Chat struct {
	q *gen.Queries
}

func NewChat(s gen.DBTX) *Chat {
	q := gen.New(s)

	return &Chat{
		q: q,
	}
}

func (c *Chat) Create(ctx context.Context, chat *models.Chat) error {
	title := pgtype.Text{}

	if err := title.Scan(chat.Title); err != nil {
		return err
	}

	desc := pgtype.Text{}
	if err := desc.Scan(chat.Description); err != nil {
		return err
	}

	_, err := c.q.CreateChat(ctx, gen.CreateChatParams{
		ChatID:      chat.ChatID,
		Title:       title,
		Type:        chat.Type,
		Description: desc,
		IsActive:    chat.IsActive,
	})

	return err
}

func (c *Chat) GetByTitle(ctx context.Context, title string) (*models.Chat, error) {
	pgTitle := pgtype.Text{}
	if err := pgTitle.Scan(title); err != nil {
		return nil, err
	}

	chat, err := c.q.GetChatByTitle(ctx, pgTitle)
	if err != nil {
		return nil, err
	}

	retChat := chatFromGenModel(chat)

	return &retChat, nil
}

func (c *Chat) GetAll(ctx context.Context) ([]models.Chat, error) {
	chats, err := c.q.GetAllChats(ctx)
	if err != nil {
		return nil, err
	}

	retChats := make([]models.Chat, 0, len(chats))

	for _, c := range chats {
		retChats = append(retChats, chatFromGenModel(c))
	}

	return retChats, nil
}

func (c *Chat) GetAllActive(ctx context.Context) ([]models.Chat, error) {
	chats, err := c.q.GetAllActiveChats(ctx)
	if err != nil {
		return nil, err
	}

	retChats := make([]models.Chat, 0, len(chats))

	for _, c := range chats {
		retChats = append(retChats, chatFromGenModel(c))
	}

	return retChats, nil
}

func (c *Chat) Delete(ctx context.Context, chatID int64) error {
	err := c.q.DeleteChatByID(ctx, chatID)

	return err
}

func chatFromGenModel(c gen.Chat) models.Chat {
	return models.Chat{
		ID:          int(c.ID),
		ChatID:      c.ChatID,
		Title:       c.Title.String,
		Type:        c.Type,
		Description: c.Description.String,
		IsActive:    c.IsActive,
	}
}
