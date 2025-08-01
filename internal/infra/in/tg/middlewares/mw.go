package middlewares

import (
	"context"
	"log/slog"
	"support_bot/internal/models"
	"support_bot/internal/pkg/logger"

	"gopkg.in/telebot.v4"
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
		userID := c.Sender().ID
		ctx := logger.AppendCtx(context.Background(), slog.Any("userID", userID))

		role, err := mw.userPr.IsAllowed(ctx, userID)
		//nolint:nilerr
		if err != nil || role == models.Denied {
			mw.l.InfoContext(ctx, "unauthorized access attempt", slog.Any("from", c.Sender()))

			return nil // Не выдаём ошибку пользователю
		}

		c.Set("isAdmin", role)

		return next(c)
	}
}

func (mw *Mw) AdminAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID

		ctx := logger.AppendCtx(context.Background(), slog.Any("userID", userID))

		role, err := mw.userPr.IsAllowed(ctx, userID)

		if err != nil || role == models.Denied || role == models.UserRole {
			mw.l.InfoContext(ctx, "unauthorized admin access attempt", slog.Any("from", c.Sender()))

			return nil // Не выдаём ошибку пользователю
		}

		// Добавляем роль в контекст, если юзер есть
		c.Set("isAdmin", role)

		return next(c)
	}
}
