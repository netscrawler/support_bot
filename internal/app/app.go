package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"support_bot/internal/collector"
	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/config"
	maxadp "support_bot/internal/delivery/max"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/delivery/telegram"
	"support_bot/internal/evaluator"
	eventcreator "support_bot/internal/event_creator"
	"support_bot/internal/generator"
	maxbot "support_bot/internal/max_bot"
	"support_bot/internal/models"
	"support_bot/internal/orchestrator"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/pkg/retry"
	"support_bot/internal/postgres"
	"support_bot/internal/processor"
	"support_bot/internal/processor/lua"
	luastd "support_bot/internal/processor/lua/stdlib"
	"support_bot/internal/processor/pipeline"
	"support_bot/internal/sheduler"
	tgbot "support_bot/internal/tg_bot"
	"support_bot/internal/tg_bot/handlers"
	"support_bot/internal/tg_bot/middlewares"
	"support_bot/internal/tg_bot/repository"
	"support_bot/internal/tg_bot/service"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
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
	Retry        *retry.Retry
}

type telegramBot struct {
	Bot        *telego.Bot
	BotHandler *th.BotHandler
	Router     *tgbot.Router
	Shed       *sheduler.SheduleAPI
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

	// go func() {
	//	err := b..Start()
	//	panic(err)
	// }()
	b.Router.Start()
}

func (b *telegramBot) stop() {
	slog.Info("stop bot polling")

	b.Router.Stop(context.TODO())
	// b.Bot.StopPoll(context.Background(), )
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

	tgBot, tHandler, err := tgbot.NewTelegramBot(ctx,
		cfg.TgBot,
		log,
	)
	if err != nil {
		return err
	}

	maxBot, err := maxbot.New(ctx, a.cfg.MaxBot, log)
	if err != nil && a.cfg.MaxBot.Enabled {
		return err
	}

	shdAPI := make(chan sheduler.SheduleAPIEvent, 5)

	mb := metabase.New(cfg.MetabaseDomain)
	appM := appmetrica.NewCollector(&cfg.AppMetrica, log)
	jiraColl := jira.New(cfg.Jira)

	sup, err := appM.GetApplications(ctx)
	if err != nil {
		log.ErrorContext(
			ctx,
			"error getting available applications for app metrica collector",
			slog.Any("error", err),
		)
	} else {
		log.InfoContext(
			ctx,
			"get available apps for collect data from app metrica",
			slog.Any("apps", sup),
		)
	}

	clct := collector.NewCollector(parallel, mb, appM, jiraColl, log)

	retr := retry.New(retry.Config{
		QueueSize:  100,
		Workers:    4,
		MaxRetries: 3,
		Backoff: retry.ExponentialBackoff{
			Base: 3 * time.Second,
			Max:  30 * time.Second,
		},
		Policy: retry.PolicyAlways{},
		Logger: log,
		Silent: false,
	})

	tg := telegram.NewChatAdaptor(tgBot, retr, log)
	maxAdp := maxadp.New(maxBot, retr, a.cfg.MaxBot.Enabled, log)

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

	eval, err := evaluator.NewEngine()
	if err != nil {
		return err
	}

	luaStdColl := luastd.NewCollector(map[string]luastd.DirectCollector{
		"jira":       jiraColl,
		"mb":         mb,
		"appmetrica": appM,
	})

	scriptRepo := lua.NewRepository(rdb.GetConn())

	luaManager := lua.NewManager(
		&cfg.Lua,
		scriptRepo,
		luastd.NewSTD(luaStdColl, luastd.DatabasePlugin{}, luastd.RateLimit{}),
	)

	runnerReg := processor.NewReg()
	luaRunner := pipeline.NewLuaRunner(luaManager)
	runnerReg.Register("lua", luaRunner)
	runnerReg.Register("sql", &pipeline.SqlRunner{})

	proc := processor.NewProcessor(runnerReg, log)

	snd := models.NewSenderProvider(tg, smbS, smtpS, maxAdp)

	delRepo := generator.NewResultRepository(rdb.GetConn(), log)

	deleter := generator.NewDeleter(delChan, tg, maxAdp, *delRepo, log)
	gen := generator.New(reportChan, clct, *snd, *delRepo, proc, eval, 4, log)

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
		Retry:        retr,
	}

	state := handlers.NewState(cfg.TgBot.CleanUpTime)
	//
	chatRepo := repository.NewChatRepository(rdb.GetConn(), log)
	userRepo := repository.NewUserRepository(rdb.GetConn(), log)
	reportRepo := repository.NewReportRepository(rdb.GetConn(), log)
	//
	notify := service.NewNotify(tg, userRepo, log)

	chatService := service.NewChat(chatRepo, notify, log)
	userService := service.NewUser(userRepo, log)
	//
	shed := sheduler.NewSheduleAPI(shdAPI)
	reportService := service.NewReportService(shed, evAPI, reportRepo, cfg.MetabaseDomain, log)

	adminHandler := handlers.NewAdminHandler(
		tgBot,
		userService,
		chatService,
		reportService,
		state,
	)
	//
	userHandler := handlers.NewUserHandler(
		tgBot,
		chatService,
		userService,
		reportService,
		state,
	)
	//
	textHandler := handlers.NewTextHandler(adminHandler, &userHandler, state)
	//
	mw := middlewares.NewMw(userService)
	//
	router := tgbot.NewRouter(tgBot, tHandler, adminHandler, &userHandler, textHandler, mw)
	//
	tgBotUser := &telegramBot{
		Bot:        tgBot,
		BotHandler: tHandler,
		Router:     router,
		Shed:       shed,
	}

	a.report = report
	a.tgBot = tgBotUser

	return nil
}
