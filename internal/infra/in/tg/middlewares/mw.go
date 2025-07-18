package middlewares

import (
	"context"
	"support_bot/internal/models"

	"gopkg.in/telebot.v4"
)

type UserProvider interface {
	IsAllowed(ctx context.Context, id int64) (string, error)
}

type Mw struct {
	userPr UserProvider
}

func NewMw(uPr UserProvider) *Mw {
	return &Mw{
		userPr: uPr,
	}
}

func (mw *Mw) UserAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID

		role, err := mw.userPr.IsAllowed(context.TODO(), userID)
		//nolint:nilerr
		if err != nil || role == models.Denied {
			return nil // Не выдаём ошибку пользователю
		}

		c.Set("isAdmin", role)

		return next(c)
	}
}

func (mw *Mw) TextAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID

		role, err := mw.userPr.IsAllowed(context.TODO(), userID)
		//nolint:nilerr
		if err != nil || role == models.Denied {
			return nil // Не выдаём ошибку пользователю
		}

		c.Set("isAdmin", role)

		return next(c)
	}
}

func (mw *Mw) AdminAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID

		role, err := mw.userPr.IsAllowed(context.TODO(), userID)
		//nolint:nilerr
		if err != nil || role == models.Denied || role == models.UserRole {
			return nil // Не выдаём ошибку пользователю
		}

		// Добавляем роль в контекст, если юзер есть
		c.Set("isAdmin", role)

		return next(c)
	}
}
