package middlewares

import (
	"context"
	"log/slog"

	"gopkg.in/telebot.v4"
	"support_bot/internal/models"
	"support_bot/internal/pkg/logger"
)

type UserProvider interface {
	IsAllowed(ctx context.Context, id int64) (string, error)
}

type Mw struct {
	userPr UserProvider
	l      *slog.Logger
}

func NewMw(uPr UserProvider) *Mw {
	l := slog.Default()

	return &Mw{
		l:      l,
		userPr: uPr,
	}
}

func (mw *Mw) UserAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		var user *telebot.User

		user = c.Sender()
		if c.Query() != nil {
			user = c.Query().Sender
		}

		ctx := logger.AppendCtx(context.Background(),
			slog.Any("userID", user.ID),
			slog.Any("username", user.Username),
			slog.Any("name", user.FirstName),
			slog.Any("from_id", c.Chat().ID),
			slog.Any("chat_name", c.Chat().Title),
		)

		role, err := mw.userPr.IsAllowed(ctx, user.ID)

		if err != nil || role == models.Denied {
			mw.l.InfoContext(
				ctx,
				"unauthorized access attempt",
			)

			if err != nil {
				mw.l.InfoContext(ctx, "error check user", slog.Any("error", err))
			}

			return nil // Не выдаём ошибку пользователю
		}

		c.Set("role", role)

		return next(c)
	}
}

func (mw *Mw) AdminAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		var user *telebot.User

		user = c.Sender()
		if c.Query() != nil {
			user = c.Query().Sender
		}

		ctx := logger.AppendCtx(context.Background(),
			slog.Any("userID", user.ID),
			slog.Any("username", user.Username),
			slog.Any("name", user.FirstName),
			slog.Any("from_id", c.Chat().ID),
			slog.Any("chat_name", c.Chat().Title),
		)

		role, err := mw.userPr.IsAllowed(ctx, user.ID)

		if err != nil || role == models.Denied || role == models.UserRole {
			mw.l.InfoContext(
				ctx,
				"unauthorized admin access attempt",
			)

			if err != nil {
				mw.l.InfoContext(ctx, "error check user", slog.Any("error", err))
			}

			return nil // Не выдаём ошибку пользователю
		}

		// Добавляем роль в контекст, если юзер есть
		c.Set("role", role)

		return next(c)
	}
}
