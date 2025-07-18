package pgrepo

import (
	"context"
	"support_bot/internal/models"

	chatrepo "support_bot/internal/infra/out/pg/gen/chat"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Chat struct {
	q *chatrepo.Queries
}

func NewChat(s *pgx.Conn) *Chat {
	q := chatrepo.New(s)
	return &Chat{
		q: q,
	}
}

func (c *Chat) Create(ctx context.Context, chat *models.Chat) error {
	title := pgtype.Text{}
	title.Scan(chat.Title)

	desc := pgtype.Text{}
	desc.Scan(chat.Description)

	_, err := c.q.CreateChat(ctx, chatrepo.CreateChatParams{
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
	pgTitle.Scan(title)

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

func (c *Chat) Delete(ctx context.Context, chatID int64) error {
	err := c.q.DeleteChatByID(ctx, chatID)
	return err
}

func chatFromGenModel(c chatrepo.Chat) models.Chat {
	return models.Chat{
		ID:          int(c.ID),
		ChatID:      c.ChatID,
		Title:       c.Title.String,
		Type:        c.Type,
		Description: c.Description.String,
		IsActive:    c.IsActive,
	}
}
