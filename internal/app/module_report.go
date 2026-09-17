package app

import (
	"context"
	"log/slog"
	"support_bot/internal/collector"
	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/jira"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/config"
	maxadp "support_bot/internal/delivery/max"
	"support_bot/internal/delivery/telegram"
	"support_bot/internal/evaluator"
	eventcreator "support_bot/internal/event_creator"
	"support_bot/internal/generator"
	"support_bot/internal/models"
	"support_bot/internal/orchestrator"
	"support_bot/internal/postgres"
	"support_bot/internal/processor"
	"support_bot/internal/processor/lua"
	luastd "support_bot/internal/processor/lua/stdlib"
	"support_bot/internal/processor/pipeline"
	reportsvc "support_bot/internal/service"
	"support_bot/internal/sheduler"
	reportstore "support_bot/internal/store"

	"go.uber.org/fx"
)

const reportBufferSize uint8 = 15

var reportPipelineModule = fx.Module("report_pipeline", fx.Provide(
	fx.Annotate(newScheduleEventsChan, fx.ResultTags(`name:"scheduleEvents"`)),
	fx.Annotate(newEventChan, fx.ResultTags(`name:"eventChan"`)),
	fx.Annotate(newDelChan, fx.ResultTags(`name:"delChan"`)),
	newSpecialEventChan,
	newShdAPIChan,
	newCronStore,
	fx.Annotate(newSheduler, fx.ParamTags(``, ``, ``, `name:"scheduleEvents"`, ``)),
	fx.Annotate(
		newEventCreator,
		fx.ParamTags(``, ``, ``, `name:"scheduleEvents"`, `name:"eventChan"`),
	),
	fx.Annotate(newEventAPI, fx.ParamTags(`name:"eventChan"`, ``)),
	newEvaluator,
	newLuaStdCollector,
	newLuaManager,
	newProcessorReg,
	newProcessor,
	newSentMsgStore,
	fx.Annotate(newDeleter, fx.ParamTags(``, `name:"delChan"`, ``, ``, ``, ``)),
	newGenerator,
	newReportStore,
	fx.Annotate(
		newOrchestrator,
		fx.ParamTags(``, `name:"eventChan"`, ``, `name:"delChan"`, ``, ``, ``, ``, ``),
	),
	newReportGenService,
))

func newScheduleEventsChan() chan models.Event {
	return make(chan models.Event, reportBufferSize)
}

func newEventChan() chan models.Event {
	return make(chan models.Event, reportBufferSize)
}

func newDelChan() chan models.Event {
	return make(chan models.Event, reportBufferSize)
}

func newSpecialEventChan() chan models.SpecialEventForLK {
	return make(chan models.SpecialEventForLK, reportBufferSize)
}

func newShdAPIChan() chan sheduler.SheduleAPIEvent {
	return make(chan sheduler.SheduleAPIEvent, 5)
}

func newSheduler(
	appCtx context.Context,
	cronStore *reportstore.CronStore,
	log *slog.Logger,
	scheduleEvents chan models.Event,
	shdAPI chan sheduler.SheduleAPIEvent,
	lc fx.Lifecycle,
) *sheduler.Sheduler {
	shd := sheduler.NewSheduler(cronStore, log, scheduleEvents, shdAPI)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error { return shd.Start(appCtx) },
		OnStop: func(context.Context) error {
			shd.Stop()

			return nil
		},
	})

	return shd
}

func newEventCreator(
	appCtx context.Context,
	cronStore *reportstore.CronStore,
	log *slog.Logger,
	scheduleEvents chan models.Event,
	eventChan chan models.Event,
	lc fx.Lifecycle,
) *eventcreator.EventCreator {
	evC := eventcreator.New(scheduleEvents, eventChan, log, cronStore)

	// No OnStop here — mirrors the original app.go asymmetry: the scheduler
	// and other lifecycle-managed components stop, but the event creator's
	// goroutine was never wired to a Stop hook in the pre-fx code either.
	lc.Append(fx.Hook{OnStart: func(context.Context) error { return evC.Start(appCtx) }})

	return evC
}

func newCronStore(rdb *postgres.DB) *reportstore.CronStore {
	return reportstore.NewCronStore(rdb.GetConn())
}

func newEventAPI(
	eventChan chan models.Event,
	specialEventChan chan models.SpecialEventForLK,
) *eventcreator.EventAPI {
	return eventcreator.NewEventAPI(eventChan, specialEventChan)
}

func newEvaluator() (*evaluator.Engine, error) {
	return evaluator.NewEngine()
}

func newLuaStdCollector(
	mb *metabase.Metabase,
	appM *appmetrica.Collector,
	jiraColl *jira.Collector,
) *luastd.CollectPlugin {
	return luastd.NewCollector(map[string]luastd.DirectCollector{
		"jira":       jiraColl,
		"mb":         mb,
		"appmetrica": appM,
	})
}

func newLuaManager(
	cfg *config.Config,
	luaStdColl *luastd.CollectPlugin,
	rdb *postgres.DB,
) *lua.Manager {
	scriptRepo := lua.NewRepository(rdb.GetConn())

	return lua.NewManager(
		&cfg.Lua,
		scriptRepo,
		luastd.NewSTD(luaStdColl, luastd.DatabasePlugin{}, luastd.RateLimit{}),
	)
}

func newProcessorReg(luaManager *lua.Manager) *processor.RunnerRegistry {
	reg := processor.NewReg()
	luaRunner := pipeline.NewLuaRunner(luaManager)
	reg.Register("lua", luaRunner)
	reg.Register("sql", &pipeline.SqlRunner{})

	return reg
}

func newProcessor(runnerReg *processor.RunnerRegistry, log *slog.Logger) *processor.Processor {
	return processor.NewProcessor(runnerReg, log)
}

func newSentMsgStore(rdb *postgres.DB, log *slog.Logger) *reportstore.SentMsgStore {
	return reportstore.NewSentMsgStore(rdb.GetConn(), log)
}

func newDeleter(
	appCtx context.Context,
	delChan chan models.Event,
	tg *telegram.ChatAdaptor,
	maxAdp *maxadp.Adaptor,
	delRepo *reportstore.SentMsgStore,
	log *slog.Logger,
	lc fx.Lifecycle,
) *orchestrator.Deleter {
	deleter := orchestrator.NewDeleter(delChan, tg, maxAdp, delRepo, log)

	lc.Append(fx.Hook{OnStart: func(context.Context) error {
		deleter.Start(appCtx)

		return nil
	}})

	return deleter
}

func newGenerator(
	appCtx context.Context,
	clct *collector.Collector,
	proc *processor.Processor,
	eval *evaluator.Engine,
	log *slog.Logger,
	lc fx.Lifecycle,
) *generator.Generator {
	gen := generator.New(clct, proc, eval, 4, log)

	lc.Append(fx.Hook{OnStart: func(context.Context) error {
		gen.Start(appCtx)

		return nil
	}})

	return gen
}

func newReportStore(rdb *postgres.DB, log *slog.Logger) *reportstore.ReportStore {
	return reportstore.NewReportStore(rdb.GetConn(), log)
}

func newOrchestrator(
	appCtx context.Context,
	eventChan chan models.Event,
	specialEventChan chan models.SpecialEventForLK,
	delChan chan models.Event,
	orchRepo *reportstore.ReportStore,
	gen *generator.Generator,
	snd *models.SenderProvider,
	delRepo *reportstore.SentMsgStore,
	log *slog.Logger,
	lc fx.Lifecycle,
) *orchestrator.Orchestrator {
	orch := orchestrator.New(
		eventChan,
		specialEventChan,
		delChan,
		orchRepo,
		gen,
		*snd,
		delRepo,
		log,
	)

	lc.Append(fx.Hook{OnStart: func(context.Context) error {
		orch.Start(appCtx)

		return nil
	}})

	return orch
}

func newReportGenService(
	reportStore *reportstore.ReportStore,
	orch *orchestrator.Orchestrator,
	log *slog.Logger,
) *reportsvc.Report {
	return reportsvc.NewReport(reportStore, orch, log)
}
