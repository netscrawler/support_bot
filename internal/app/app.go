package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"support_bot/internal/collector"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/config"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/delivery/telegram"
	"support_bot/internal/evaluator"
	eventcreator "support_bot/internal/event_creator"
	"support_bot/internal/generator"
	"support_bot/internal/models"
	"support_bot/internal/orchestrator"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/postgres"
	"support_bot/internal/sheduler"
	bot "support_bot/internal/tg_bot"
	"support_bot/internal/tg_bot/handlers"
	"support_bot/internal/tg_bot/middlewares"
	"support_bot/internal/tg_bot/repository"
	"support_bot/internal/tg_bot/service"

	"golang.org/x/net/proxy"
	"gopkg.in/telebot.v4"
)

const (
	parallel          uint8 = 30
	channelBufferSize uint8 = 15
)

type app struct {
	ctx    context.Context
	cancel context.CancelFunc

	log     *slog.Logger
	storage *postgres.DB
	cfg     *config.Config
	report  *reportApp

	tgBot *telegramBot
	smb   *smb.SMB
}

type reportApp struct {
	ScheduleC    chan models.Event
	EventC       chan models.Event
	Scheduler    *sheduler.Sheduler
	Event        *eventcreator.EventCreator
	Orchestrator *orchestrator.Orchestrator
	Generator    *generator.Generator
	Deleter      *generator.Deleter
}

type telegramBot struct {
	Bot    *telebot.Bot
	Router *bot.Router
	Shed   *sheduler.SheduleAPI
}

func New(ctx context.Context, cfg *config.Config) (*app, error) {
	appCtx, cancelApp := context.WithCancel(ctx)
	log := slog.Default()

	app := &app{
		ctx:    appCtx,
		cancel: cancelApp,
		log:    log,
		cfg:    cfg,
	}

	if err := app.init(appCtx); err != nil {
		cancelApp()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			cfg.Database.DatabaseConnect,
		)
		defer cancel()

		return nil, errors.Join(err, app.close(shutdownCtx))
	}

	return app, nil
}

func (a *app) Start(_ context.Context) error {
	a.tgBot.start()

	return a.report.start(a.ctx)
}

func (a *app) GracefulShutdown(ctx context.Context) {
	log := a.log
	log.InfoContext(ctx, "start")

	if err := a.close(ctx); err != nil {
		log.ErrorContext(ctx, "unable to stop app correctly", slog.Any("error", err))

		return
	}

	log.InfoContext(ctx, "successfully stop")
}

func (a *app) close(ctx context.Context) error {
	a.cancel()

	if a.tgBot != nil {
		a.tgBot.stop()
	}

	if a.report != nil {
		a.report.stop(ctx)
	}

	var err error

	if a.smb != nil {
		err = errors.Join(err, a.smb.Close())
	}

	if a.storage != nil {
		err = errors.Join(err, a.storage.Stop(ctx))
	}

	return err
}

func (r *reportApp) start(ctx context.Context) error {
	err := r.Scheduler.Start(ctx)
	if err != nil {
		return err
	}

	err = r.Event.Start(ctx)
	if err != nil {
		return err
	}

	r.Generator.Start(ctx)
	r.Deleter.Start(ctx)
	r.Orchestrator.Start(ctx)

	return nil
}

func (r *reportApp) stop(_ context.Context) {
	r.Scheduler.Stop()
}

func (b *telegramBot) start() {
	slog.Info("starting bot polling")

	go b.Bot.Start()
}

func (b *telegramBot) stop() {
	slog.Info("stop bot polling")

	b.Bot.Stop()
}

func (a *app) init(ctx context.Context) error {
	log := a.log
	cfg := a.cfg

	connCtx, cancel := context.WithTimeout(ctx, cfg.Database.DatabaseConnect)
	defer cancel()

	connCtx = logger.AppendCtx(connCtx, slog.Any("function", "connecting to database"))

	rdb, err := postgres.New(connCtx, cfg.Database, log)
	if err != nil {
		log.ErrorContext(connCtx, "unable to create connection", slog.Any("error", err))

		return err
	}

	a.storage = rdb

	tgBot, err := newTelegramClient(
		cfg.Bot.TelegramToken,
		cfg.Bot.Proxy,
		cfg.Bot.ApiProxy,
		cfg.Bot.BotPoll,
		log,
	)
	if err != nil {
		return err
	}

	shdAPI := make(chan sheduler.SheduleAPIEvent, 5)

	mb := metabase.New(cfg.MetabaseDomain)
	clct := collector.NewCollector(parallel, mb, log)

	tg := telegram.NewChatAdaptor(tgBot, log)
	smtpS := smtp.New(cfg.SMTP, log)

	var smbS *smb.SMB

	smbS, err = smb.New(
		ctx,
		cfg.SMB,
		log,
	)
	if err != nil {
		return err
	}

	a.smb = smbS

	sheduleEvents := make(chan models.Event, channelBufferSize)
	eventChan := make(chan models.Event, channelBufferSize)
	delChan := make(chan models.Event, channelBufferSize)
	reportChan := make(chan models.Report, channelBufferSize)
	specialEventChan := make(chan models.SpecialEventForLK, channelBufferSize)

	shdLoader := sheduler.NewSheduleRepo(rdb.GetConn(), log)
	shd := sheduler.NewSheduler(shdLoader, log, sheduleEvents, shdAPI)

	evRepository := eventcreator.NewRepository(rdb.GetConn(), log)
	evC := eventcreator.New(sheduleEvents, eventChan, log, evRepository)
	evAPI := eventcreator.NewEventAPI(eventChan, specialEventChan)

	eval, err := evaluator.NewEvaluator()
	if err != nil {
		return err
	}

	snd := models.NewSenderProvider(tg, smbS, smtpS)

	delRepo := generator.NewResultRepository(rdb.GetConn(), log)

	deleter := generator.NewDeleter(delChan, tg, *delRepo, log)
	gen := generator.New(reportChan, clct, *snd, *delRepo, eval, 4, log)

	orchRepo := orchestrator.NewRepository(rdb.GetConn(), log)
	orch := orchestrator.New(eventChan, specialEventChan, reportChan, delChan, orchRepo, log)
	report := &reportApp{
		ScheduleC:    sheduleEvents,
		EventC:       eventChan,
		Scheduler:    shd,
		Event:        evC,
		Orchestrator: orch,
		Generator:    gen,
		Deleter:      deleter,
	}

	state := handlers.NewState(cfg.Bot.CleanUpTime)

	chatRepo := repository.NewChatRepository(rdb.GetConn(), log)
	userRepo := repository.NewUserRepository(rdb.GetConn(), log)
	reportRepo := repository.NewReportRepository(rdb.GetConn(), log)

	notify := service.NewNotify(tg, userRepo, log)

	chatService := service.NewChat(chatRepo, notify, log)
	userService := service.NewUser(userRepo, log)

	shed := sheduler.NewSheduleAPI(shdAPI)
	reportService := service.NewReportService(shed, evAPI, reportRepo, log)

	adminHandler := handlers.NewAdminHandler(
		tgBot,
		userService,
		chatService,
		reportService,
		state,
	)

	userHandler := handlers.NewUserHandler(
		tgBot,
		chatService,
		userService,
		reportService,
		state,
	)

	textHandler := handlers.NewTextHandler(adminHandler, userHandler, state)

	mw := middlewares.NewMw(userService)

	router := bot.NewRouter(tgBot, adminHandler, userHandler, textHandler, mw)

	router.Setup()
	tgBotUser := &telegramBot{
		Bot:    tgBot,
		Router: router,
		Shed:   shed,
	}

	a.report = report
	a.tgBot = tgBotUser

	return nil
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
			Timeout: 30 * time.Second,
		}, nil

	case "socks5", "socks5h":
		dialer, err := proxy.SOCKS5("tcp", u.Host, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}

		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				},
			},
			Timeout: 30 * time.Second,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
}

func newTelegramClient(
	token string,
	proxy string,
	apiProxy string,
	poll time.Duration,
	log *slog.Logger,
) (*telebot.Bot, error) {
	client := &http.Client{}
	if proxy != "" {
		log.Info(
			"proxy addr not empty, creating bot with system proxy",
			slog.Any("proxy addr", proxy),
		)
		clientB, err := buildHTTPClient(proxy)
		if err != nil {
			return nil, err
		}
		client = clientB
	}

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: poll},
		Client: client,
	}

	if apiProxy != "" {
		log.Info(
			"api proxy not empty, creating bot with custom api server",
			slog.Any("api server", apiProxy),
		)
		pref.URL = apiProxy
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return b, nil
}
