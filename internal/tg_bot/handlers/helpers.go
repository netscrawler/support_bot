package handlers

import (
	"fmt"

	"support_bot/internal/models"

	tele "gopkg.in/telebot.v4"
)

func mapReportRPLToMarkup(rp models.LoadReportRPL) tele.ReplyMarkup {
	var rows [][]tele.InlineButton

	for _, report := range rp.Reports {
		rows = append(rows, []tele.InlineButton{
			{
				Unique: "report",
				Text:   report.Title,
				Data:   fmt.Sprintf("%d;%s;%d", report.ID, report.Name, rp.CurrentPage),
			},
		})
	}

	var back, curr, next tele.InlineButton

	if rp.CurrentPage > 1 {
		back = tele.InlineButton{
			Unique: "back_report_list",
			Text:   "Back",
			Data:   fmt.Sprintf("%d", rp.CurrentPage-1),
		}
	}

	if rp.CurrentPage < rp.PageCount {
		next = tele.InlineButton{
			Unique: "next_report_list",
			Text:   "Next",
			Data:   fmt.Sprintf("%d", rp.CurrentPage+1),
		}
	}

	curr = tele.InlineButton{
		Unique: "_",
		Text:   fmt.Sprintf("%d/%d", rp.CurrentPage, rp.PageCount),
	}

	navRow := make([]tele.InlineButton, 0, 3)

	if back.Unique != "" {
		navRow = append(navRow, back)
	}

	navRow = append(navRow, curr)

	if next.Unique != "" {
		navRow = append(navRow, next)
	}

	rows = append(rows, navRow)

	return tele.ReplyMarkup{InlineKeyboard: rows}
}

func getMarkupForReport(r models.ReportInfo, pageFrom int) tele.ReplyMarkup {
	rows := [][]tele.InlineButton{
		{
			{
				Unique: "back_to_report_list",
				Text:   "Назад",
				Data:   fmt.Sprintf("%d", pageFrom),
			},
			{
				Unique: "report_resend",
				Text:   "Запустить",
				Data:   fmt.Sprintf("%s;%s", r.ID, r.Name),
			},
			{
				Unique: "report_get",
				Text:   "Получить",
				Data:   fmt.Sprintf("%s;%s", r.ID, r.Name),
			},
		},
	}

	return tele.ReplyMarkup{InlineKeyboard: rows}
}
