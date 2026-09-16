package app

import (
	"context"
	"log/slog"
	"support_bot/internal/config"
	"support_bot/internal/postgres"
	"support_bot/internal/sheduler"
	reportstore "support_bot/internal/store"
	"support_bot/internal/tg_bot/handlers"
	"support_bot/internal/tg_bot/middlewares"
	"support_bot/internal/tg_bot/repository"
	"support_bot/internal/tg_bot/service"

	"support_bot/internal/delivery/telegram"
	eventcreator "support_bot/internal/event_creator"
	maxbot "support_bot/internal/max_bot"
	tgbot "support_bot/internal/tg_bot"

	maxcli "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"go.uber.org/fx"
)

var telegramModule = fx.Module(
	"telegram",
	fx.Provide(
		newTelegramBot,
		newMaxBot,
		newTgState,
		newChatRepository,
		newUserRepository,
		newNotify,
		newChatService,
		newUserService,
		newScheduleAPI,
		newReportService,
		newAdminHandler,
		newUserHandler,
		newTextHandler,
		newMw,
		newRouter,
	),
)

func newTelegramBot(ctx context.Context, cfg *config.Config, log *slog.Logger) (*telego.Bot, *th.BotHandler, error) {
	return tgbot.NewTelegramBot(ctx, cfg.TgBot, log)
}

func newMaxBot(ctx context.Context, cfg *config.Config, log *slog.Logger) (*maxcli.Api, error) {
	maxBot, err := maxbot.New(ctx, cfg.MaxBot, log)
	if err != nil && cfg.MaxBot.Enabled {
		return nil, err
	}

	return maxBot, nil
}

func newTgState(cfg *config.Config) *handlers.State {
	return handlers.NewState(cfg.TgBot.CleanUpTime)
}

func newChatRepository(rdb *postgres.DB, log *slog.Logger) *repository.ChatRepository {
	return repository.NewChatRepository(rdb.GetConn(), log)
}

func newUserRepository(rdb *postgres.DB, log *slog.Logger) *repository.UserRepository {
	return repository.NewUserRepository(rdb.GetConn(), log)
}

func newNotify(tg *telegram.ChatAdaptor, userRepo *repository.UserRepository, log *slog.Logger) *service.Notify {
	return service.NewNotify(tg, userRepo, log)
}

func newChatService(chatRepo *repository.ChatRepository, notify *service.Notify, log *slog.Logger) *service.Chat {
	return service.NewChat(chatRepo, notify, log)
}

func newUserService(userRepo *repository.UserRepository, log *slog.Logger) *service.User {
	return service.NewUser(userRepo, log)
}

func newScheduleAPI(shdAPI chan sheduler.SheduleAPIEvent) *sheduler.SheduleAPI {
	return sheduler.NewSheduleAPI(shdAPI)
}

func newReportService(
	shed *sheduler.SheduleAPI,
	evAPI *eventcreator.EventAPI,
	reportStore *reportstore.ReportStore,
	cfg *config.Config,
	log *slog.Logger,
) *service.Report {
	return service.NewReportService(shed, evAPI, reportStore, cfg.MetabaseDomain, log)
}

func newAdminHandler(
	tgBot *telego.Bot,
	userService *service.User,
	chatService *service.Chat,
	reportService *service.Report,
	state *handlers.State,
) *handlers.AdminHandler {
	return handlers.NewAdminHandler(tgBot, userService, chatService, reportService, state)
}

func newUserHandler(
	tgBot *telego.Bot,
	chatService *service.Chat,
	userService *service.User,
	reportService *service.Report,
	state *handlers.State,
) *handlers.UserHandler {
	userHandler := handlers.NewUserHandler(tgBot, chatService, userService, reportService, state)

	return &userHandler
}

func newTextHandler(
	adminHandler *handlers.AdminHandler,
	userHandler *handlers.UserHandler,
	state *handlers.State,
) *handlers.TextHandler {
	return handlers.NewTextHandler(adminHandler, userHandler, state)
}

func newMw(userService *service.User) *middlewares.Mw {
	return middlewares.NewMw(userService)
}

func newRouter(
	tgBot *telego.Bot,
	tHandler *th.BotHandler,
	adminHandler *handlers.AdminHandler,
	userHandler *handlers.UserHandler,
	textHandler *handlers.TextHandler,
	mw *middlewares.Mw,
	lc fx.Lifecycle,
) *tgbot.Router {
	router := tgbot.NewRouter(tgBot, tHandler, adminHandler, userHandler, textHandler, mw)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			router.Start()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return router.Stop(ctx)
		},
	})

	return router
}
