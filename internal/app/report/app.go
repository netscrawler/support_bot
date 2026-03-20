package report

import (
	"context"
	"log/slog"
	"support_bot/internal/config"
	"support_bot/internal/core"
	"support_bot/internal/core/actions/builtin"
	"support_bot/internal/core/actions/builtin/collector"
	"support_bot/internal/core/actions/builtin/collector/metabase"
	"support_bot/internal/delivery"
	"support_bot/internal/delivery/smb"
	"support_bot/internal/delivery/smtp"
	"support_bot/internal/delivery/telegram"
	"support_bot/internal/postgres"

	"gopkg.in/telebot.v4"

	models "support_bot/internal/models/report"
)

const (
	parralell         uint8 = 30
	channelBufferSize uint8 = 15
)

type App struct {
	sheduleC  chan string
	eventC    chan string
	sheduler  *core.Sheduler
	event     *core.EventCreator
	orch      *core.Orchestrator
	generator *core.Generator
	delivery  *delivery.SenderStrategy

	log *slog.Logger
}

func New(
	ctx context.Context,
	cfg *config.Config,
	bot *telebot.Bot,
	db *postgres.DB,
	shdAPI chan core.SheduleAPIEvent,
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

	shdLoader := core.NewSheduleRepo(db.GetConn(), log)
	shd := core.NewSheduler(shdLoader, log, sheduleEvents, shdAPI)

	evRepository := core.NewEventRepository(db.GetConn(), log)
	evC := core.NewEventCreator(sheduleEvents, eventChan, log, evRepository)

	eval, err := builtin.NewEvaluator(log)
	if err != nil {
		return nil, err
	}

	delivery := delivery.NewSender(tg, smb, smtpS, log)

	generator := core.New(reportChan, clct, delivery, eval, 4, log)

	orchRepo := core.NewOrchestratorRepository(db.GetConn(), log)
	orch := core.NewOrchestrator(eventChan, reportChan, orchRepo, log)

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
