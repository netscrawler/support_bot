package bot

import (
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

func NewRouter(bot *telebot.Bot, ahl *handlers.AdminHandler, uhl *handlers.UserHandler) *Router {
	return &Router{
		bot:     bot,
		adminHl: ahl,
		userHl:  uhl,
	}
}

func (r *Router) Setup() {
	r.bot.Use(middleware.Logger())
	adminOnly := r.bot.Group()
	// FIX: Убрать хардкод админа
	// TODO: добавить обработку админа
	adminOnly.Use(middleware.Whitelist(476788912))
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
	adminOnly.Handle(telebot.OnText, r.adminHl.ProcessUserInput)
	adminOnly.Handle(&menu.SendNotifyAdmin, r.adminHl.SendNotification)

	// TODO: настроить обработку юзера
	// userOnly := r.bot.Group()
	// userOnly.Use(middleware.Whitelist(0))
	// userOnly.Handle(menu.StartCommand, r.userHl.StartUser)
	// userOnly.Handle(&menu.SendNotifyUser, nil)
	// 	userOnly := r.bot.Group()
	// 	userOnly.Use(middleware.Whitelist(0))
	// 	userOnly.Handle(&menu.SendNotifyUser, nil)
	//
	// 	// TODO: добавить обработку регистрации
	// 	register := r.bot.Group()
	// 	// register.Use(nil)
	// 	register.Handle(menu.StartCommand, r.adminHl.StartAdmin)
}
