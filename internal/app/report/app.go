package report

import (
	"context"
	"log/slog"
	"support_bot/internal/collector"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/config"
	"support_bot/internal/delivery"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/delivery/telegram"
	"support_bot/internal/evaluator"
	"support_bot/internal/generator"
	"support_bot/internal/orchestrator"
	"support_bot/internal/postgres"
	"support_bot/internal/sheduler"

	"gopkg.in/telebot.v4"

	eventcreator "support_bot/internal/event_creator"

	models "support_bot/internal/models/report"
)

const (
	parralell         uint8 = 30
	channelBufferSize uint8 = 15
)

type App struct {
	sheduleC  chan string
	eventC    chan string
	sheduler  *sheduler.Sheduler
	event     *eventcreator.EventCreator
	orch      *orchestrator.Orchestrator
	generator *generator.Generator
	delivery  *delivery.SenderStrategy

	log *slog.Logger
}

func New(
	ctx context.Context,
	cfg *config.Config,
	bot *telebot.Bot,
	db *postgres.DB,
	shdAPI chan sheduler.SheduleAPIEvent,
	log *slog.Logger,
) (*App, error) {
	mb := metabase.New(cfg.MetabaseDomain)
	clct := collector.NewCollector(parralell, mb, log)

	tg := telegram.NewChatAdaptor(bot, log)
	smtpS := smtp.New(cfg.SMTP, log)

	smb, err := smb.New(
		ctx,
		cfg.SMB,
		log,
	)
	if err != nil {
		return nil, err
	}

	sheduleEvents := make(chan string, channelBufferSize)
	eventChan := make(chan string, channelBufferSize)
	reportChan := make(chan models.Report, channelBufferSize)

	shdLoader := sheduler.NewSheduleRepo(db.GetConn(), log)
	shd := sheduler.NewSheduler(shdLoader, log, sheduleEvents, shdAPI)

	evRepository := eventcreator.NewRepository(db.GetConn(), log)
	evC := eventcreator.New(sheduleEvents, eventChan, log, evRepository)

	eval, err := evaluator.NewEvaluator(log)
	if err != nil {
		return nil, err
	}

	delivery := delivery.NewSender(tg, smb, smtpS, log)

	generator := generator.New(reportChan, clct, delivery, eval, 16, log)

	orchRepo := orchestrator.NewRepository(db.GetConn(), log)
	orch := orchestrator.New(eventChan, reportChan, orchRepo, log)

	return &App{
		sheduleC:  sheduleEvents,
		eventC:    eventChan,
		sheduler:  shd,
		event:     evC,
		orch:      orch,
		log:       log,
		generator: generator,
		delivery:  delivery,
	}, nil
}

func (r *App) Start(ctx context.Context) error {
	err := r.sheduler.Start(ctx)
	if err != nil {
		return err
	}

	err = r.event.Start(ctx)
	if err != nil {
		return err
	}

	r.generator.Start(ctx)

	r.orch.Start(ctx)

	return nil
}

func (r *App) Stop(_ context.Context) {
	r.sheduler.Stop()
	close(r.sheduleC)
	close(r.eventC)
}
