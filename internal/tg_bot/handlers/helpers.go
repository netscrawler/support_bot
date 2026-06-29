package handlers

import (
	"fmt"

	tele "gopkg.in/telebot.v4"
	"support_bot/internal/models"
)

func mapReportRPLToMarkup(rp models.LoadReportRPL) tele.ReplyMarkup {
	var rows [][]tele.InlineButton

	for _, report := range rp.Reports {
		rows = append(rows, []tele.InlineButton{
			{
				Unique: "report",
				Text:   report.Title,
				Data:   fmt.Sprintf("%d;%s", report.ID, report.Name),
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
