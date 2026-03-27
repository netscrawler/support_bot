package bot

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"support_bot/internal/core"
	"support_bot/internal/delivery/telegram"
	"support_bot/internal/postgres"
	"support_bot/internal/tg_bot/handlers"
	"support_bot/internal/tg_bot/middlewares"
	"support_bot/internal/tg_bot/repository"
	"support_bot/internal/tg_bot/service"
	"time"

	"gopkg.in/telebot.v4"

	bot "support_bot/internal/tg_bot"

	"golang.org/x/net/proxy"
)

type Bot struct {
	bot    *telebot.Bot
	router *bot.Router

	shed *core.SheduleAPI
}

func buildHTTPClient(proxyStr string) (*http.Client, error) {
	if proxyStr == "" {
		return &http.Client{}, nil
	}

	u, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {

	case "http", "https":
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(u),
			},
			Timeout: 120 * time.Second,
		}, nil

	case "socks5", "socks5h":
		dialer, err := proxy.SOCKS5("udp", u.Host, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
			Timeout: 120 * time.Second,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
}

func NewTgBot(
	token string,
	urlProxy string,
	proxy string,
	poll time.Duration,
) (*telebot.Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poll},
	}
	if proxy != "" {
		client, err := buildHTTPClient(proxy)
		if err != nil {
			return nil, err
		}
		pref.Client = client
	}

	if urlProxy != "" {
		pref.URL = urlProxy
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func New(
	cleanupTime time.Duration,
	tgBot *telebot.Bot,
	db *postgres.DB,
	shdAPI chan core.SheduleAPIEvent,
	log *slog.Logger,
) (*Bot, error) {
	state := handlers.NewState(cleanupTime)

	chatRepo := repository.NewChatRepository(db.GetConn(), log)
	userRepo := repository.NewUserRepository(db.GetConn(), log)

	chatService := service.NewChat(chatRepo, log)
	userService := service.NewUser(userRepo, log)

	tgSender := telegram.NewChatAdaptor(tgBot, log)
	shed := core.NewSheduleAPI(shdAPI)
	reportService := service.NewReportService(shed, log)

	notifyier := service.NewTelegramNotify(userRepo, chatRepo, tgSender, log)

	adminHandler := handlers.NewAdminHandler(
		tgBot,
		userService,
		chatService,
		notifyier,
		reportService,
		state,
	)

	userHandler := handlers.NewUserHandler(
		tgBot,
		chatService,
		userService,
		state,
		notifyier,
	)

	textHandler := handlers.NewTextHandler(adminHandler, userHandler, state)

	mw := middlewares.NewMw(userService)

	router := bot.NewRouter(tgBot, adminHandler, userHandler, textHandler, mw)

	router.Setup()

	return &Bot{
		bot:    tgBot,
		router: router,
		shed:   shed,
	}, nil
}

func (b *Bot) Start() {
	slog.Info("starting bot polling")

	go b.bot.Start()
}

func (b *Bot) Stop() {
	slog.Info("stop bot polling")

	b.shed.StopAPI()
}
