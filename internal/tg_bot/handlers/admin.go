package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"support_bot/internal/models"
	"support_bot/internal/tg_bot/menu"
	"support_bot/internal/tg_bot/service"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

type AdminHandler struct {
	bot         *telego.Bot
	userService *service.User
	chatService *service.Chat
	report      *service.Report
	state       *State
}

func NewAdminHandler(
	bot *telego.Bot,
	userService *service.User,
	chatService *service.Chat,
	report *service.Report,
	state *State,
) *AdminHandler {
	return &AdminHandler{
		bot:         bot,
		userService: userService,
		chatService: chatService,
		state:       state,
		report:      report,
	}
}

func (h *AdminHandler) Start(ctx *th.Context, message telego.Message) error {
	return showAdminMenu(ctx.Bot(), message)
}

func (h *AdminHandler) ListReports(ctx *th.Context, query telego.CallbackQuery) error {
	if err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	}); err != nil {
		return err
	}

	tctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rpl, err := h.report.LoadReportsWithPagination(tctx)
	if err != nil {
		return editOrSend(ctx, query, "Ошибка получения отчетов", nil)
	}

	mark := mapReportRPLToAdminMarkup(rpl)

	err = editOrSend(ctx, query, menu.MsgHelloReport, &mark)

	return err
}

func (h *AdminHandler) LoadReportPage(ctx *th.Context, query telego.CallbackQuery) error {
	data := strings.Split(query.Data, ";")
	if len(data) != 2 {
		return h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось определить страницу",
		})
	}
	page, err := strconv.Atoi(data[1])
	if err != nil {
		return h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось определить страницу",
		})
	}

	if err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	}); err != nil {
		return err
	}

	tctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rpl, err := h.report.LoadReportByPage(tctx, page)
	if err != nil {
		_, err = h.bot.EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: query.Message.GetChat().ID},
			MessageID: query.Message.GetMessageID(),
			Text:      "Ошибка получения отчетов: " + err.Error(),
		})
		return err
	}

	mark := mapReportRPLToAdminMarkup(rpl)

	err = editOrSend(ctx, query, menu.MsgHelloReport, &mark)

	return err
}

func (h *AdminHandler) IgnoreReportPage(ctx *th.Context, query telego.CallbackQuery) error {
	err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	return err
}

func (h *AdminHandler) SelectReport(ctx *th.Context, query telego.CallbackQuery) error {
	repInf, err := getReportInfoFromQuery(query)
	if err != nil {
		return h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось определить отчет",
		})
	}

	if err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	}); err != nil {
		return err
	}

	tctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rpInf, err := h.report.GetReportInfoByName(tctx, repInf.ReportName)
	if err != nil {
		return h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось получить информацию об отчете",
		})
	}
	rpInf.ID = repInf.ID

	mark := getMarkupForReport(rpInf, repInf.PageFrom)

	return editOrSendRich(ctx, query, rpInf.String(), &mark)
}

func (h *AdminHandler) ResendSelectReport(ctx *th.Context, query telego.CallbackQuery) error {
	repInf, err := getReportInfoFromQuery(query)
	if err != nil {
		return h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось определить отчет",
		})
	}
	if err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	}); err != nil {
		return err
	}

	tctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	h.report.GenerateAndSendReportByName(tctx, repInf.ReportName)

	return editOrSend(ctx, query, "Отчет запущен. Результат будет отправлен получателям", nil)
}

func (h *AdminHandler) GenerateSelectedReport(
	ctx *th.Context,
	query telego.CallbackQuery,
) error {
	repInf, err := getReportInfoFromQuery(query)
	if err != nil {
		return h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось определить отчет",
		})
	}

	if err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	}); err != nil {
		return err
	}

	tctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	title := query.Message.Message().Chat.FirstName

	chat := &models.Chat{
		ChatID:   query.Message.GetChat().ID,
		Title:    &title,
		Type:     query.Message.GetChat().Type,
		IsActive: true,
		ChType:   models.ChatTypeTg,
	}

	if err := h.report.GenerateReportByName(tctx, repInf.ReportName, chat); err != nil {
		_, err = h.bot.EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: query.Message.GetChat().ID},
			MessageID: query.Message.Message().MessageID,
			Text:      "Не удалось запустить отчет",
		})
		return err
	}

	return editOrSend(ctx, query, "Отчет запущен. Результат придет в этот чат.", nil)
}

func (h *AdminHandler) ManageUsers(ctx *th.Context, query telego.CallbackQuery) error {
	rmkp := tu.InlineKeyboard(
		tu.InlineKeyboardCols(
			1,
			menu.ListUser,
			menu.AddUser,
			menu.RemoveUser,
		)...)

	return editOrSend(ctx, query, "Управление пользователями", rmkp)
}

func (h *AdminHandler) ListUsers(ctx *th.Context, query telego.CallbackQuery) error {
	users, err := h.userService.GetAll(ctx)
	if err != nil {
		return h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось получить список пользователей",
		})
	}

	return nil
}

func (h *AdminHandler) DeleteUser(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) ShowUser(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) ManageChats(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) ListChats(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) DeleteChats(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) ShowChats(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) InfoChatCommand(ctx *th.Context, message telego.Message) error {
	return nil
}

func (h *AdminHandler) ManageCrons(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) ListCrons(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) SwitchCronStatus(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) StartJobs(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func (h *AdminHandler) StopJobs(ctx *th.Context, query telego.CallbackQuery) error {
	return nil
}

func showAdminMenu(bot *telego.Bot, message telego.Message) error {
	rmkp := tu.InlineKeyboard(
		tu.InlineKeyboardCols(
			1,
			menu.ShowReportsAdmin,
			menu.ManageCron,
			menu.ManageChats,
			menu.ManageUsers,
		)...)

	_, err := bot.SendMessage(
		context.Background(),
		&telego.SendMessageParams{
			ChatID:      message.Chat.ChatID(),
			Text:        "Меню администратора",
			ReplyMarkup: rmkp,
		},
	)

	return err
}
