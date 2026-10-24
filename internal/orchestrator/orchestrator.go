package orchestrator

import (
	"context"
	"errors"
	"html"
	"log/slog"
	"maps"
	"time"

	"support_bot/internal/generator"
	"support_bot/internal/models"
	"support_bot/internal/pkg/logger"
	"support_bot/internal/pkg/secret"
)

// ponytail: семафор фиксированного размера, не связан с реальным числом
// воркеров генератора (захардкожено 4 в app.go, само по себе должно
// переехать под fx) — синхронизируйте значения или прокиньте это через
// New(), если потребуется тонкая настройка пропускной способности.
const (
	maxInFlightGenerations = 8
)

type ReportLoader interface {
	LoadActive(ctx context.Context) ([]models.Report, error)
	LoadByEvent(
		ctx context.Context,
		event string,
		active bool,
	) (*models.Report, error)
}

// SentMsgSaver сохраняет информацию об отправленном сообщении, чтобы deleter
// мог позже его удалить (например, сообщения в Telegram в конце дня).
type SentMsgSaver interface {
	SaveTgMsg(
		ctx context.Context,
		reportName string,
		msgs []models.SentMessage,
	) error
}

// RunRecorder записывает итог запуска отчёта в журнал (internal/runlog).
type RunRecorder interface {
	Record(ctx context.Context, run models.Run)
	LastSuccessHash(ctx context.Context, report string) (string, bool)
}

type Orchestrator struct {
	EventC        chan models.Event
	SpecialEventC chan models.SpecialEventForLK

	DeleteC chan models.Event

	rL  ReportLoader
	gen *generator.Generator

	snd         models.SenderProvider
	sentMsgRepo SentMsgSaver
	runs        RunRecorder

	genSem chan struct{}

	log *slog.Logger
}

func New(
	evC chan models.Event,
	specialEventC chan models.SpecialEventForLK,
	delC chan models.Event,
	rl ReportLoader,
	gen *generator.Generator,
	snd models.SenderProvider,
	sentMsgRepo SentMsgSaver,
	runs RunRecorder,
	log *slog.Logger,
) *Orchestrator {
	l := log.With(slog.Any("module", "orchestrator"))

	return &Orchestrator{
		EventC:        evC,
		SpecialEventC: specialEventC,
		DeleteC:       delC,
		rL:            rl,
		gen:           gen,
		snd:           snd,
		sentMsgRepo:   sentMsgRepo,
		runs:          runs,
		genSem:        make(chan struct{}, maxInFlightGenerations),
		log:           l,
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.log.InfoContext(ctx, "starting...")

	go o.run(ctx)
}

// Generate реализует однофайловую сигнатуру Generate из
// service.ReportGenerator (internal/service/report_generator.go), пропуская
// разовый (on-demand) запрос через тот же самый шаг generate, что и
// событийный путь — оркестратор не различает "отправить" и "вернуть",
// разница только в том, что делается с результатом. Разовые отчеты должны
// объявлять ровно один экспорт; если их несколько, возвращается первый —
// осознанное упрощение. Отрицательный результат оценки или отсутствие
// экспортов транслируется в models.ErrNotFound — так, как это ожидает
// HTTP-слой ("не найдено").
func (o *Orchestrator) Generate(
	ctx context.Context,
	report models.Report,
) (models.Data, error) {
	_, data, approve, err := o.generate(ctx, report)
	if err != nil {
		return models.Data{}, err
	}

	if !approve || len(data) == 0 {
		return models.Data{}, models.ErrNotFound
	}

	return data[0], nil
}

func (o *Orchestrator) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			o.log.InfoContext(ctx, "context cancelled. stopping")

			return

		case event, ok := <-o.EventC:
			if !ok {
				o.log.WarnContext(ctx, "event chan closed")

				return
			}

			switch event.Type {
			case models.EventTypeDeleteSentReport:
				o.processDelReportEvent(ctx, event.Name)

			default:
				o.processGenReportEvent(
					ctx,
					event.Name,
					true,
					nil,
					models.RunKindCron,
					"",
				)
			}

		case event, ok := <-o.SpecialEventC:
			if !ok {
				o.log.WarnContext(ctx, "event chan closed")

				return
			}

			switch event.Event.Type {
			case models.EventTypeGenReportForTG:
				kind := models.RunKindLK
				if event.Test {
					kind = models.RunKindTest
				}

				o.processGenReportEvent(
					ctx,
					event.Event.Name,
					false,
					&event.Recipient,
					kind,
					event.Period,
				)

			case models.EventTypeGenReport:
				o.processGenReportEvent(
					ctx,
					event.Event.Name,
					false,
					nil,
					models.RunKindManual,
					"",
				)

			default:
			}
		}
	}
}

// processGenReportEvent загружает отчет(ы), зарегистрированные на event, и
// доставляет каждый из них — вызывающий код не различает, какой канал или
// тип события это вызвал, кроме двух реально варьируемых параметров:
// затрагивает ли событие только активные отчеты, и опциональный override
// получателя для разового случая (например, "сгенерировать прямо сейчас для
// этого чата в ЛК").
func (o *Orchestrator) processGenReportEvent(
	ctx context.Context,
	event string,
	active bool,
	recipientOverride *models.Recipient,
	kind models.RunKind,
	period models.PeriodName,
) {
	reports, err := o.getReportByEvent(ctx, event, active)
	if err != nil {
		o.log.ErrorContext(ctx, "error loading report", slog.Any("error", err))

		return
	}

	for _, report := range reports {
		var chatID *int64

		// Пробный запуск админа экспортирует отчёт даже при ложном evaluate.
		report.IgnoreEvaluate = kind == models.RunKindTest
		report.Period = period

		if recipientOverride != nil {
			report.Recipients = []models.Recipient{*recipientOverride}

			if recipientOverride.Chat != nil {
				chatID = &recipientOverride.Chat.ChatID
			}
		}

		o.log.DebugContext(
			ctx,
			"sending report to generator",
			slog.Any("report", report.Name),
		)

		select {
		case o.genSem <- struct{}{}:
			go func(report models.Report) {
				defer func() { <-o.genSem }()

				o.generateAndDeliver(ctx, report, kind, chatID)
			}(report)

		case <-ctx.Done():
			return
		}
	}
}

// generate прогоняет отчет через общий пул воркеров генератора в контексте
// логирования. Бюджет таймаута на один отчет — забота самого пула воркеров
// (internal/generator/generator.go), здесь не дублируется. Функция
// намеренно не делает ничего с результатом — передать его дальше, доставив
// (generateAndDeliver) или вернув разовому вызывающему (Generate), решают
// вызывающие функции ниже.
func (o *Orchestrator) generate(
	ctx context.Context,
	report models.Report,
) (models.Dataset, []models.Data, bool, error) {
	ctx = logger.AppendCtx(ctx, slog.Any("report_name", report.Name))

	return o.gen.Generate(ctx, report)
}

// generateAndDeliver прогоняет отчет через общий пул воркеров генератора и,
// при положительной оценке, доставляет его получателям отчета. Запускается
// в отдельной горутине, чтобы занятый пул никогда не блокировал event loop.
// Итог запуска пишется в журнал ровно один раз.
func (o *Orchestrator) generateAndDeliver(
	ctx context.Context,
	report models.Report,
	kind models.RunKind,
	chatID *int64,
) {
	run := models.Run{
		ReportName: report.Name,
		Kind:       kind,
		Status:     models.RunStatusSuccess,
		ChatID:     chatID,
		StartedAt:  time.Now(),
	}

	defer func() {
		run.FinishedAt = time.Now()
		o.runs.Record(ctx, run)
	}()

	dataset, data, approve, err := o.generate(ctx, report)
	if err != nil {
		o.log.ErrorContext(ctx, "error create report", slog.Any("error", err))

		run.Status, run.Error = models.RunStatusFailed, err.Error()
		o.notifyTest(ctx, kind, chatID, errText(err))

		return
	}

	if !approve {
		if kind != models.RunKindTest {
			run.Status, run.Reason = models.RunStatusSkipped, "evaluate"

			return
		}

		// Пробный запуск доставляется и при ложном условии.
		run.Reason = "evaluate_false"
	}

	if kind != models.RunKindTest && kind != models.RunKindLK {
		hash := dataset.Hash()

		run.DataHash = hash

		if hash == "" {
			o.log.WarnContext(
				ctx,
				"dataset не маршалится, дедупликация не применяется",
				slog.String("report", report.Name),
			)
		}

		if report.Dedup && kind == models.RunKindCron && hash != "" {
			if prev, ok := o.runs.LastSuccessHash(
				ctx,
				report.Name,
			); ok &&
				prev == hash {
				o.log.InfoContext(
					ctx,
					"данные не изменились, отправка пропущена",
					slog.String("report", report.Name),
				)

				run.Status, run.Reason = models.RunStatusSkipped, models.RunReasonDuplicate

				return
			}
		}
	}

	delivered, err := o.deliver(ctx, report, dataset, data)
	if err != nil {
		o.log.ErrorContext(
			ctx,
			"error delivering report",
			slog.Any("error", err),
		)

		run.Status, run.Error = models.RunStatusFailed, err.Error()
		if delivered > 0 {
			run.Status = models.RunStatusPartial
		}

		o.notifyTest(ctx, kind, chatID, errText(err))

		return
	}

	evalLine := "🧪 evaluate = true"
	if !approve {
		evalLine = "🧪 evaluate = false — по расписанию отчёт не был бы отправлен"
	}

	o.notifyTest(ctx, kind, chatID, evalLine)
}

// errText — текст ошибки для чата: до 500 символов (по рунам), с экранированием HTML.
func errText(err error) string {
	// Redact до обрезки, иначе обрезка может разрезать секрет.
	r := []rune(secret.Redact(err.Error()))
	if len(r) > 500 {
		r = r[:500]
	}

	return "❌ " + html.EscapeString(string(r))
}

// notifyTest шлёт служебную строку в чат админа при пробном запуске.
func (o *Orchestrator) notifyTest(
	ctx context.Context,
	kind models.RunKind,
	chatID *int64,
	text string,
) {
	if kind != models.RunKindTest || chatID == nil {
		return
	}

	if _, err := o.snd.Tg().
		SendText(ctx, models.TgChat{ChatID: *chatID}, text); err != nil {
		o.log.WarnContext(
			ctx,
			"test run note send failed",
			slog.Any("error", err),
		)
	}
}

// deliver отправляет сгенерированный отчет настроенным получателям и
// сохраняет состояние отправленных сообщений. Возвращает число получателей,
// которым отправка прошла без ошибки, и ошибку отправки (если была).
func (o *Orchestrator) deliver(
	ctx context.Context,
	report models.Report,
	data models.Dataset,
	res []models.Data,
) (int, error) {
	l := o.log

	if len(report.Recipients) == 0 {
		l.ErrorContext(ctx, "empty targets list")

		return 0, errors.New("empty targets list")
	}

	// Достаем "_meta" лист из данных, для использования в шаблоне email
	addMeta := make(map[string]any)
	meta, ok := data["_meta"]

	if ok {
		for _, d := range meta {
			maps.Insert(addMeta, maps.All(d))
		}
	}

	l.InfoContext(
		ctx,
		"meta",
		slog.Any("meta", addMeta),
		slog.Any("_meta", data["_meta"]),
	)

	msg := models.NewMessage(report.Name, res, addMeta, report.Recipients...)

	resMsg, delivered, sendErr := msg.Send(ctx, o.snd)
	if sendErr != nil {
		l.ErrorContext(
			ctx,
			"error while send message",
			slog.Any("error", sendErr),
		)
	}

	if len(resMsg) == 0 {
		l.InfoContext(ctx, "report generated")

		return delivered, sendErr
	}

	l.InfoContext(
		ctx,
		"saving message to database",
		slog.Any("report", report.Name),
		slog.Any("message", resMsg),
	)

	if err := o.sentMsgRepo.SaveTgMsg(ctx, msg.ReportName, resMsg); err != nil {
		l.WarnContext(ctx, "result msg save failed", slog.Any("error", err))
	}

	return delivered, sendErr
}

func (o *Orchestrator) processDelReportEvent(
	ctx context.Context,
	event string,
) {
	select {
	case <-ctx.Done():
		o.log.InfoContext(ctx, "context cancelled. stopping")

		return

	case o.DeleteC <- models.Event{Name: event, Type: models.EventTypeDeleteSentReport}:
		o.log.InfoContext(ctx, "sending delete event to deleter")
	}
}

func (o *Orchestrator) getReportByEvent(
	ctx context.Context,
	event string,
	active bool,
) ([]models.Report, error) {
	l := o.log.With(slog.Any("event", event))
	l.DebugContext(ctx, "getting report by event")

	l.DebugContext(ctx, "cache miss, loading report")

	reports, err := o.rL.LoadByEvent(ctx, event, active)
	if err != nil {
		l.ErrorContext(
			ctx,
			"error while loading report",
			slog.Any("error", err),
		)

		return nil, err
	}

	if !active {
		return []models.Report{*reports}, nil
	}

	l.DebugContext(ctx, "reports loaded", slog.Any("reports_count", 1))

	return []models.Report{*reports}, nil
}
