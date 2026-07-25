package handlers

import (
	"context"
	"log/slog"
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

type UserHandler struct {
	bot         *telego.Bot
	chatService *service.Chat
	userService *service.User
	report      *service.Report
	state       *State

	log *slog.Logger
}

func NewUserHandler(
	bot *telego.Bot,
	chatService *service.Chat,
	userService *service.User,
	reportService *service.Report,
	state *State,
) UserHandler {
	return UserHandler{
		bot:         bot,
		chatService: chatService,
		userService: userService,
		report:      reportService,
		state:       state,
	}
}

func showUserMenu(bot *telego.Bot, message telego.Message) error {
	rmkp := tu.InlineKeyboard(
		tu.InlineKeyboardCols(1, menu.ShowReports)...)

	_, err := bot.SendMessage(
		context.Background(),
		&telego.SendMessageParams{
			ChatID:      message.Chat.ChatID(),
			Text:        "Здравствуйте! Я помогу быстро получить нужные отчёты.\n\nНажмите кнопку ниже, чтобы выбрать отчёт и отправить его в этот чат.",
			ReplyMarkup: rmkp,
		},
	)

	return err
}

const helpMsg = `
Вот что можно сделать:
• /start — открыть главное меню
• Нажмите «📊 Выбрать отчёт» и выберите нужный вариант
• После запуска отчёт будет отправлен в этот чат
`

func (u *UserHandler) Start(ctx *th.Context, message telego.Message) error {
	return showUserMenu(ctx.Bot(), message)
}

func (u *UserHandler) Help(ctx *th.Context, message telego.Message) error {
	// u.log.Info("User help command from %s", message.From.Username)

	_, err := ctx.Bot().SendMessage(ctx.Context(), &telego.SendMessageParams{
		ChatID: tu.ID(message.Chat.ID),
		Text:   helpMsg,
	})
	if err != nil {
		u.log.Error("Error", slog.Any("error", err))

		return err
	}

	return nil
}

// Back returns user to main user menu when "back" callback is pressed
func (u *UserHandler) Back(ctx *th.Context, query telego.CallbackQuery) error {
	_ = u.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})

	rmkp := tu.InlineKeyboard(
		tu.InlineKeyboardCols(1, menu.ShowReports)...)

	return editOrSend(ctx, query, "Здравствуйте! Я помогу быстро получить нужные отчёты.\n\nНажмите кнопку ниже, чтобы выбрать отчёт и отправить его в этот чат.", &rmkp)
}

func (h *UserHandler) LoadReportsPage(ctx *th.Context, query telego.CallbackQuery) error {
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

	mark := mapReportRPLToMarkup(rpl)

	err = editOrSend(ctx, query, menu.MsgHelloReport, &mark)

	return err
}

func (h *UserHandler) IgnoreReportPage(ctx *th.Context, query telego.CallbackQuery) error {
	err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	return err
}

func (h *UserHandler) GenerateSelectedReport(
	ctx *th.Context,
	query telego.CallbackQuery,
) error {
	data := strings.Split(query.Data, ";")
	if len(data) < 2 {
		err := h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Не удалось определить отчет",
		})
		return err
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

	if err := h.report.GenerateReportByName(tctx, data[2], chat); err != nil {
		_, err = h.bot.EditMessageText(ctx, &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: query.Message.GetChat().ID},
			MessageID: query.Message.Message().MessageID,
			Text:      "Не удалось запустить отчёт. Повторите попытку позже.",
		})
		return err
	}

	return editOrSend(ctx, query, "Отчёт поставлен в очередь. Результат придёт в этот чат в течение нескольких минут.", nil)
}

func (h *UserHandler) LoadReports(ctx *th.Context, query telego.CallbackQuery) error {
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

	mark := mapReportRPLToMarkup(rpl)

	err = editOrSend(ctx, query, menu.MsgHelloReport, &mark)

	return err
}
