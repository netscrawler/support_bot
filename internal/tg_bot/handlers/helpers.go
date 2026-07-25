package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"support_bot/internal/models"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/mymmrac/telego/telegoutil"
)

func mapReportRPLToMarkup(rp models.LoadReportRPL) telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton

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

func mapReportRPLToAdminMarkup(rp models.LoadReportRPL) telego.InlineKeyboardMarkup {
	var rows [][]telego.InlineKeyboardButton

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

	rows = append(rows, navRow)

	return telego.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func getMarkupForReport(r models.ReportInfo, pageFrom int) telego.InlineKeyboardMarkup {
	return telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{
					Text:         "⬅️ Назад",
					CallbackData: fmt.Sprintf("back_to_report_list;%d", pageFrom),
				},
				{
					Text:         "▶️ Запустить",
					CallbackData: fmt.Sprintf("report_resend;%s;%s", r.ID, r.Name),
				},
				{
					Text:         "📤 Получить",
					CallbackData: fmt.Sprintf("report_get;%s;%s;%d", r.ID, r.Name, pageFrom),
				},
			},
		},
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
		_ = ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})

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

	_ = ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

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
		_ = ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})

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

	_ = ctx.Bot().AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	return nil
}

type reportInfoFromQuery struct {
	ID         string
	ReportName string
	PageFrom   int
	Callback   string
}

func getReportInfoFromQuery(query telego.CallbackQuery) (reportInfoFromQuery, error) {
	data := strings.Split(query.Data, ";")
	if len(data) != 4 {
		return reportInfoFromQuery{}, fmt.Errorf(
			"report info must be a format: (callback;id;report_name;page_from) %s", query.Data,
		)
	}

	callback, id, reportName, pageFromStr := data[0], data[1], data[2], data[3]
	if reportName == "" {
		return reportInfoFromQuery{}, fmt.Errorf("empty report name from query")
	}

	pageFrom, err := strconv.Atoi(pageFromStr)
	if err != nil {
		pageFrom = 0
	}

	_, err = strconv.Atoi(id)
	if err != nil {
		return reportInfoFromQuery{}, fmt.Errorf("report id from query must be a integer")
	}

	return reportInfoFromQuery{
		ID:         id,
		ReportName: reportName,
		PageFrom:   pageFrom,
		Callback:   callback,
	}, nil
}
