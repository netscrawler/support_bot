package handlers

import (
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"support_bot/internal/models"
	"support_bot/internal/tg_bot/menu"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
)

const (
	maxCallbackData = 64
	runErrMaxRunes  = 200
)

func mapReportRPLToMarkup(rp models.LoadReportRPL) telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(rp.Reports)+1)

	for _, report := range rp.Reports {
		rows = append(rows, []telego.InlineKeyboardButton{
			{
				Text: report.Title,
				CallbackData: fmt.Sprintf(
					"report_gen;%d;%s;%d",
					report.ID,
					report.Name,
					rp.CurrentPage,
				),
			},
		})
	}

	navRow := make([]telego.InlineKeyboardButton, 0, 3)

	if rp.CurrentPage > 1 {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "⬅️",
			CallbackData: fmt.Sprintf("user_report_page;%d", rp.CurrentPage-1),
		})
	}

	navRow = append(navRow, telego.InlineKeyboardButton{
		Text:         fmt.Sprintf("%d/%d", rp.CurrentPage, rp.PageCount),
		CallbackData: "ignore",
	})

	if rp.CurrentPage < rp.PageCount {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "➡️",
			CallbackData: fmt.Sprintf("user_report_page;%d", rp.CurrentPage+1),
		})
	}

	rows = append(rows, navRow)

	return telego.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func mapReportRPLToAdminMarkup(
	rp models.LoadReportRPL,
) telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(rp.Reports)+2)

	for _, report := range rp.Reports {
		rows = append(rows, []telego.InlineKeyboardButton{
			{
				Text: report.Title,
				CallbackData: fmt.Sprintf(
					"admin_report_select;%d;%s;%d",
					report.ID,
					report.Name,
					rp.CurrentPage,
				),
			},
		})
	}

	navRow := make([]telego.InlineKeyboardButton, 0, 3)

	if rp.CurrentPage > 1 {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "⬅️",
			CallbackData: fmt.Sprintf("admin_report_page;%d", rp.CurrentPage-1),
		})
	}

	navRow = append(navRow, telego.InlineKeyboardButton{
		Text:         fmt.Sprintf("%d/%d", rp.CurrentPage, rp.PageCount),
		CallbackData: "ignore",
	})

	if rp.CurrentPage < rp.PageCount {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "➡️",
			CallbackData: fmt.Sprintf("admin_report_page;%d", rp.CurrentPage+1),
		})
	}

	rows = append(rows, navRow, []telego.InlineKeyboardButton{menu.Back})

	return telego.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func getMarkupForReport(
	r models.ReportInfo,
	pageFrom int,
) telego.InlineKeyboardMarkup {
	return telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{
					Text: msgBack,
					CallbackData: fmt.Sprintf(
						"back_to_report_list;%d",
						pageFrom,
					),
				},
				{
					Text: "▶️ Запустить",
					CallbackData: fmt.Sprintf(
						"report_resend;%s;%s;%d",
						r.ID,
						r.Name,
						pageFrom,
					),
				},
				{
					Text: "📤 Получить",
					CallbackData: fmt.Sprintf(
						"report_get;%s;%s;%d",
						r.ID,
						r.Name,
						pageFrom,
					),
				},
			},
			{
				{
					Text: "🧾 Запуски",
					CallbackData: fmt.Sprintf(
						"report_runs;%s;%s;%d",
						r.ID,
						r.Name,
						pageFrom,
					),
				},
			},
		},
	}
}

func answerCallback(ctx *th.Context, query telego.CallbackQuery) {
	err := ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})
	if err != nil {
		slog.Warn("answer callback query", slog.Any("error", err))
	}
}

func editOrSend(
	ctx *th.Context,
	query telego.CallbackQuery,
	text string,
	markup *telego.InlineKeyboardMarkup,
) error {
	_, err := ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:          telegoutil.ID(query.Message.GetChat().ID),
		MessageID:       query.Message.GetMessageID(),
		InlineMessageID: query.InlineMessageID,
		ParseMode:       telego.ModeHTML,
		Text:            text,
		ReplyMarkup:     markup,
	})
	if err == nil {
		answerCallback(ctx, query)

		return nil
	}

	_, sendErr := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
		ChatID:      telegoutil.ID(query.Message.GetChat().ID),
		Text:        text,
		ParseMode:   telego.ModeHTML,
		ReplyMarkup: markup,
	})
	if sendErr != nil {
		return sendErr
	}

	answerCallback(ctx, query)

	return nil
}

func editOrSendRich(
	ctx *th.Context,
	query telego.CallbackQuery,
	text string,
	markup *telego.InlineKeyboardMarkup,
) error {
	_, err := ctx.Bot().EditMessageText(ctx, &telego.EditMessageTextParams{
		ChatID:          telegoutil.ID(query.Message.GetChat().ID),
		MessageID:       query.Message.GetMessageID(),
		InlineMessageID: query.InlineMessageID,
		ReplyMarkup:     markup,
		RichMessage:     &telego.InputRichMessage{HTML: text},
	})
	if err == nil {
		answerCallback(ctx, query)

		return nil
	}

	_, sendErr := ctx.Bot().SendRichMessage(ctx, &telego.SendRichMessageParams{
		ChatID:      telegoutil.ID(query.Message.GetChat().ID),
		ReplyMarkup: markup,
		RichMessage: telego.InputRichMessage{HTML: text},
	})
	if sendErr != nil {
		return sendErr
	}

	answerCallback(ctx, query)

	return nil
}

type reportInfoFromQuery struct {
	ID         string
	ReportName string
	PageFrom   int
	Callback   string
	// PeriodCode — необязательное 5-е поле callback (t|y|w|m).
	PeriodCode string
}

func getReportInfoFromQuery(
	query telego.CallbackQuery,
) (reportInfoFromQuery, error) {
	data := strings.Split(query.Data, ";")
	if len(data) != 4 && len(data) != 5 {
		return reportInfoFromQuery{}, fmt.Errorf(
			"report info must be a format: (callback;id;report_name;page_from[;period]) %s",
			query.Data,
		)
	}

	periodCode := ""
	if len(data) == 5 {
		periodCode = data[4]
	}

	callback, id, reportName, pageFromStr := data[0], data[1], data[2], data[3]
	if reportName == "" {
		return reportInfoFromQuery{}, errors.New("empty report name from query")
	}

	pageFrom, err := strconv.Atoi(pageFromStr)
	if err != nil {
		pageFrom = 0
	}

	_, err = strconv.Atoi(id)
	if err != nil {
		return reportInfoFromQuery{}, errors.New(
			"report id from query must be a integer",
		)
	}

	return reportInfoFromQuery{
		ID:         id,
		ReportName: reportName,
		PageFrom:   pageFrom,
		Callback:   callback,
		PeriodCode: periodCode,
	}, nil
}

// periodStep решает, что делать с запуском: вернуть markup — показать выбор периода,
// иначе запускать за period (пустой — DefaultPeriod отчёта).
func periodStep(
	rep reportInfoFromQuery,
	usesPeriod bool,
	back string,
) (models.PeriodName, *telego.InlineKeyboardMarkup) {
	if rep.PeriodCode != "" {
		p, ok := models.PeriodByCode(rep.PeriodCode)
		if !ok {
			slog.Warn(
				"unknown period code, using default",
				slog.String("data", rep.PeriodCode),
			)
		}

		return p, nil
	}

	if !usesPeriod {
		return "", nil
	}

	base := fmt.Sprintf(
		"%s;%s;%s;%d;",
		rep.Callback,
		rep.ID,
		rep.ReportName,
		rep.PageFrom,
	)
	// Telegram: callback data не длиннее 64 байт; код всегда один символ.
	if len(base)+1 > maxCallbackData {
		slog.Warn("period callback too long, using default period",
			slog.String("report", rep.ReportName))

		return "", nil
	}

	btn := func(text string, p models.PeriodName) telego.InlineKeyboardButton {
		return telego.InlineKeyboardButton{
			Text:         text,
			CallbackData: base + p.Code(),
		}
	}

	return "", &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				btn("Сегодня", models.PeriodToday),
				btn("Вчера", models.PeriodYesterday),
			},
			{
				btn("Неделя (7 дн.)", models.PeriodWeek),
				btn("Прошлый месяц", models.PeriodMonth),
			},
			{{Text: msgBack, CallbackData: back}},
		},
	}
}

// periodPhrase — « за вчера» для текста подтверждения; для пустого периода — "".
func periodPhrase(p models.PeriodName) string {
	switch p {
	case models.PeriodToday:
		return " за сегодня"
	case models.PeriodYesterday:
		return " за вчера"
	case models.PeriodWeek:
		return " за неделю"
	case models.PeriodMonth:
		return " за прошлый месяц"
	default:
		return ""
	}
}

var runStatusIcon = map[models.RunStatus]string{
	models.RunStatusSuccess: "✅",
	models.RunStatusSkipped: "⏭",
	models.RunStatusFailed:  "❌",
	models.RunStatusPartial: "⚠️",
}

func formatRuns(name string, runs []models.Run, err error) string {
	head := fmt.Sprintf(
		"🧾 Последние запуски <b>%s</b>",
		html.EscapeString(name),
	)

	switch {
	case err != nil:
		return head + "\n\nНе удалось получить журнал запусков."
	case len(runs) == 0:
		return head + "\n\nЗапусков пока нет."
	}

	var sb strings.Builder

	sb.WriteString(head + "\n")

	for _, r := range runs {
		sb.WriteString("\n" + formatRun(r))
	}

	return sb.String()
}

func formatRun(r models.Run) string {
	icon := runStatusIcon[r.Status]
	line := fmt.Sprintf(
		"%s · %s · %s %s · %s",
		r.StartedAt.Local().Format("02.01 15:04"),
		html.EscapeString(string(r.Kind)),
		icon,
		html.EscapeString(string(r.Status)),
		r.FinishedAt.Sub(r.StartedAt).Round(100*time.Millisecond),
	)

	detail := r.Error
	if detail == "" {
		detail = r.Reason
	}

	if detail == "" {
		return line
	}

	if rs := []rune(detail); len(rs) > runErrMaxRunes {
		detail = string(rs[:runErrMaxRunes]) + "…"
	}

	return line + " · " + html.EscapeString(detail)
}
