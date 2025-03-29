package bot

import (
	"context"
	"log"
	"support_bot/internal/bot/handlers"
	"support_bot/internal/bot/menu"
	"support_bot/internal/models"

	"gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

type UserProvider interface {
	IsAllowed(ctx context.Context, id int64) (string, error)
}

type Router struct {
	bot     *telebot.Bot
	adminHl *handlers.AdminHandler
	textHl  *handlers.TextHandler
	userHl  *handlers.UserHandler
	userPr  UserProvider
}

func NewRouter(
	bot *telebot.Bot,
	ahl *handlers.AdminHandler,
	uhl *handlers.UserHandler,
	thl *handlers.TextHandler,
	userPr UserProvider,
) *Router {
	return &Router{
		bot:     bot,
		adminHl: ahl,
		userHl:  uhl,
		textHl:  thl,
		userPr:  userPr,
	}
}

func (r *Router) Setup() {
	r.bot.Use(middleware.Logger())

	// Группа для регистрации (если юзер не в базе)
	register := r.bot.Group()
	register.Handle(menu.RegisterCommand, r.userHl.RegisterUser)

	text := r.bot.Group()
	text.Handle(telebot.OnText, r.textHl.ProcessTextInput, r.TextAuthMiddleware)

	// Группа пользователей
	userOnly := r.bot.Group()

	userOnly.Use(r.UserAuthMiddleware)

	userOnly.Handle(menu.UserStart, r.userHl.StartUser)
	userOnly.Handle(&menu.SendNotifyUser, r.userHl.SendNotification)

	// userOnly.Handle(telebot.OnText, r.textHl.ProcessTextInput)
	userOnly.Handle(
		&telebot.InlineButton{Unique: "confirm_user_notification"},
		r.adminHl.ConfirmSendNotification,
	)
	userOnly.Handle(
		&telebot.InlineButton{Unique: "cancel_user_notification"},
		r.adminHl.CancelSendNotification,
	)

	// Группа админов (должна быть зарегистрирована **после** userOnly)
	adminOnly := r.bot.Group()
	adminOnly.Use(r.AdminAuthMiddleware)
	adminOnly.Handle(menu.StartCommand, r.adminHl.StartAdmin)
	adminOnly.Handle(&menu.ManageUsers, r.adminHl.ManageUsers)
	adminOnly.Handle(&menu.ManageChats, r.adminHl.ManageChats)
	adminOnly.Handle(&menu.ListUser, r.adminHl.ListUsers)
	adminOnly.Handle(&menu.AddUser, r.adminHl.AddUser)
	adminOnly.Handle(&menu.RemoveUser, r.adminHl.RemoveUser)
	adminOnly.Handle(&menu.ListChats, r.adminHl.ListChats)
	adminOnly.Handle(menu.AddChat, r.adminHl.ProcessAddChat)
	adminOnly.Handle(&menu.RemoveChat, r.adminHl.RemoveChat)
	adminOnly.Handle(&menu.Back, r.adminHl.StartAdmin)
	// adminOnly.Handle(telebot.OnText, r.textHl.ProcessTextInput)
	adminOnly.Handle(&menu.SendNotifyAdmin, r.adminHl.SendNotification)
	// r.bot.Handle(telebot.OnText, r.textHl.ProcessTextInput)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "confirm_notification"},
		r.adminHl.ConfirmSendNotification,
	)
	adminOnly.Handle(
		&telebot.InlineButton{Unique: "cancel_notification"},
		r.adminHl.CancelSendNotification,
	)
}

func (r *Router) UserAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID
		log.Printf("UserAuthMiddleware triggered for user: %d", userID)

		// Проверяем пользователя в базе
		role, err := r.userPr.IsAllowed(context.TODO(), userID)
		if err != nil || role == models.Denied {
			log.Printf("Error checking user %d: %v, %s", userID, err, role)
			return nil // Не выдаём ошибку пользователю
		}

		// Добавляем роль в контекст, если юзер есть
		c.Set("isAdmin", role)

		log.Printf("User %d authenticated as %s", userID, role)
		return next(c)
	}
}

func (r *Router) TextAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID
		log.Printf("TextAuthMiddleware triggered for user: %d", userID)

		// Проверяем пользователя в базе
		role, err := r.userPr.IsAllowed(context.TODO(), userID)
		if err != nil || role == models.Denied {
			log.Printf("Error checking user %d: %v, %s", userID, err, role)
			return nil // Не выдаём ошибку пользователю
		}

		// Добавляем роль в контекст, если юзер есть
		c.Set("isAdmin", role)

		log.Printf("User %d authenticated as %s", userID, role)
		return next(c)
	}
}

func (r *Router) AdminAuthMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID
		role, err := r.userPr.IsAllowed(context.TODO(), userID)
		if err != nil || role == models.Denied || role == models.UserRole {
			log.Printf("Error checking user %d: %v", userID, err)
			return nil // Не выдаём ошибку пользователю
		}

		// Добавляем роль в контекст, если юзер есть
		c.Set("isAdmin", role)

		log.Printf("User %d authenticated as %s", userID, role)
		return next(c)
	}
}
