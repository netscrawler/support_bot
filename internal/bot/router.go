package bot

import (
	"context"
	"support_bot/internal/bot/handlers"
	"support_bot/internal/bot/menu"

	"gopkg.in/telebot.v4"
	"gopkg.in/telebot.v4/middleware"
)

type Router struct {
	bot     *telebot.Bot
	adminHl *handlers.AdminHandler
	userHl  *handlers.UserHandler
}

func NewRouter(ctx context.Context) *Router {
	return &Router{}
}

func (r *Router) Setup() {
	r.bot.Use(middleware.Logger())
	adminOnly := r.bot.Group()
	// FIX: Убрать хардкод админа
	adminOnly.Use(middleware.Whitelist(476788912))
	adminOnly.Handle(&menu.ManageUsers, nil)
	adminOnly.Handle(&menu.ManageChats, nil)
	adminOnly.Handle(&menu.ListUser, nil)
	adminOnly.Handle(&menu.AddUser, nil)
	adminOnly.Handle(&menu.RemoveUser, nil)
	adminOnly.Handle(&menu.ListChats, nil)
	adminOnly.Handle(&menu.AddChat, nil)
	adminOnly.Handle(&menu.RemoveChat, nil)
	adminOnly.Handle(&menu.Back, nil)
	adminOnly.Handle(&menu.SendNotifyAdmin, nil)

	userOnly := r.bot.Group()
	userOnly.Use(middleware.Whitelist(0))
	userOnly.Handle(&menu.SendNotifyUser, nil)
}
