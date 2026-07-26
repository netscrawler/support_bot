package handlers

import (
	"context"
	"errors"
	"fmt"
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
		return editOrSend(ctx, query, "Ошибка получения отчетов", menu.BackMarkup)
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
			ChatID:      telego.ChatID{ID: query.Message.GetChat().ID},
			MessageID:   query.Message.GetMessageID(),
			Text:        "Ошибка получения отчетов: " + err.Error(),
			ReplyMarkup: menu.BackMarkup,
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
			Text:            "Не удалось определить отчет" + err.Error(),
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

	return editOrSend(
		ctx,
		query,
		"Отчёт поставлен в очередь. Результат будет отправлен получателям в ближайшее время.",
		menu.BackMarkup,
	)
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
			ChatID:      telego.ChatID{ID: query.Message.GetChat().ID},
			MessageID:   query.Message.Message().MessageID,
			Text:        "Не удалось запустить отчет",
			ReplyMarkup: menu.BackMarkup,
		})
		return err
	}

	return editOrSend(
		ctx,
		query,
		"Отчёт поставлен в очередь. Результат придёт в этот чат в течение нескольких минут.",
		menu.BackMarkup,
	)
}

func (h *AdminHandler) ManageUsers(ctx *th.Context, query telego.CallbackQuery) error {
	mkp := telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{{
		menu.ListUser,
	}, {
		menu.AddUser,
		menu.RemoveUser,
	}, {
		menu.Back,
	}}}

	return editOrSend(ctx, query, "Выберите действие для управления пользователями.", &mkp)
}

func (h *AdminHandler) AddUser(ctx *th.Context, query telego.CallbackQuery) error {
	h.state.set(query.Message.GetChat().ID, addUserState)
	return editOrSend(
		ctx,
		query,
		"Введите username пользователя для добавления.\nНапример: ivan.petrov",
		nil,
	)
}

func (h *AdminHandler) RemoveUser(ctx *th.Context, query telego.CallbackQuery) error {
	h.state.set(query.Message.GetChat().ID, removeUserState)
	return editOrSend(
		ctx,
		query,
		"Введите username пользователя для удаления.\nНапример: ivan.petrov",
		nil,
	)
}

func (h *AdminHandler) ListUsers(ctx *th.Context, query telego.CallbackQuery) error {
	users, err := h.userService.GetAll(ctx.Context())
	if err != nil {
		return editOrSend(ctx, query, "Не удалось получить список пользователей", menu.BackMarkup)
	}

	if len(users) == 0 {
		return editOrSend(ctx, query, "Список пользователей пуст", menu.BackMarkup)
	}

	rows := make([][]telego.InlineKeyboardButton, 0, len(users)+1)
	for _, user := range users {
		text := user.Username
		if user.Username == "" {
			text = fmt.Sprintf("%d", user.TelegramID)
		}

		rows = append(rows, []telego.InlineKeyboardButton{{
			Text:         fmt.Sprintf("%s (%s)", text, user.Role),
			CallbackData: fmt.Sprintf("show_user;%s", user.Username),
		}})
	}

	rows = append(rows, []telego.InlineKeyboardButton{{
		Text:         "🔙 Назад",
		CallbackData: "manage_users",
	}})

	mkp := telego.InlineKeyboardMarkup{InlineKeyboard: rows}

	return editOrSend(ctx, query, "Список пользователей", &mkp)
}

func (h *AdminHandler) DeleteUser(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")
	if len(parts) == 1 || parts[1] == "" {
		users, err := h.userService.GetAll(ctx.Context())
		if err != nil {
			return editOrSend(
				ctx,
				query,
				"Не удалось получить список пользователей",
				menu.BackMarkup,
			)
		}

		rows := make([][]telego.InlineKeyboardButton, 0, len(users)+1)
		for _, user := range users {
			name := user.Username
			if name == "" {
				name = fmt.Sprintf("%d", user.TelegramID)
			}

			rows = append(rows, []telego.InlineKeyboardButton{{
				Text:         fmt.Sprintf("❌ %s", name),
				CallbackData: fmt.Sprintf("delete_user;%s", name),
			}})
		}

		rows = append(rows, []telego.InlineKeyboardButton{{
			Text:         "🔙 Назад",
			CallbackData: "manage_users",
		}})

		mkp := telego.InlineKeyboardMarkup{InlineKeyboard: rows}
		return editOrSend(ctx, query, "Выберите пользователя для удаления", &mkp)
	}

	username := parts[1]
	if err := h.userService.Delete(ctx.Context(), username, false); err != nil {
		return editOrSend(
			ctx,
			query,
			fmt.Sprintf("Не удалось удалить пользователя %s", username),
			menu.BackMarkup,
		)
	}

	return editOrSend(ctx, query, fmt.Sprintf("Пользователь %s удалён", username), menu.BackMarkup)
}

func (h *AdminHandler) ShowUser(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")
	if len(parts) != 2 {
		return editOrSend(ctx, query, "Не удалось определить пользователя", menu.BackMarkup)
	}

	user, err := h.userService.GetByUsername(ctx.Context(), parts[1])
	if err != nil {
		return editOrSend(
			ctx,
			query,
			"Не удалось получить информацию о пользователе",
			menu.BackMarkup,
		)
	}

	text := fmt.Sprintf(
		"Пользователь: %s\nID: %d\nРоль: %s",
		user.Username,
		user.TelegramID,
		user.Role,
	)
	return editOrSend(ctx, query, text, nil)
}

func (h *AdminHandler) AddChat(ctx *th.Context, message telego.Message) error {
	if message.Chat.Type != telego.ChatTypePrivate && message.Chat.Type != telego.ChatTypeGroup {
		_, err := ctx.Bot().SendMessage(ctx, &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: message.Chat.ID},
			Text:   "Добавление чата возможно только группе.",
		})
		return err
	}

	dto := models.NewTgChatDTO(
		message.Chat.ID,
		message.Chat.Title,
		message.Chat.Type,
		message.Chat.Username,
	)
	dto.DeActivate()

	tctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := h.chatService.Add(tctx, dto)
	return err
}

func (h *AdminHandler) ListChats(ctx *th.Context, query telego.CallbackQuery) error {
	chats, err := h.chatService.GetAll(ctx.Context())
	if err != nil {
		return editOrSend(
			ctx,
			query,
			"Не удалось получить список чатов"+err.Error(),
			menu.BackMarkup,
		)
	}

	if len(chats) == 0 {
		return editOrSend(ctx, query, "Список чатов пуст", menu.BackMarkup)
	}

	rows := make([][]telego.InlineKeyboardButton, 0, len(chats)+1)
	for _, chat := range chats {
		rows = append(rows, []telego.InlineKeyboardButton{{
			Text:         fmt.Sprintf("💬 %s", chat.Title),
			CallbackData: fmt.Sprintf("show_chat;%s;%d", chat.Title, chat.ID),
		}})
	}

	rows = append(rows, []telego.InlineKeyboardButton{{
		Text:         "🔙 Назад",
		CallbackData: "back",
	}})

	mkp := telego.InlineKeyboardMarkup{InlineKeyboard: rows}
	return editOrSend(ctx, query, "Список чатов", &mkp)
}

func (h *AdminHandler) DeleteChat(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")
	title := parts[1]

	mark := telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				telego.InlineKeyboardButton{
					Text:         "Да",
					CallbackData: fmt.Sprintf("confirm_delete_chat;%s", title),
				},
				telego.InlineKeyboardButton{
					Text:         "Нет",
					CallbackData: "back",
				},
			},
		},
	}

	return editOrSend(
		ctx,
		query,
		fmt.Sprintf("Вы уверены, что хотите удалить чат %s?", title),
		&mark,
	)
}

func (h *AdminHandler) ConfirmDeleteChat(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")

	if len(parts) != 2 {
		return editOrSend(ctx, query, "Не удалось определить чат", menu.BackMarkup)
	}
	title := parts[1]

	tctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()

	err := h.chatService.Remove(tctx, title)
	if err != nil {
		return editOrSend(
			ctx,
			query,
			fmt.Sprintf("Не удалось удалить чат %s: %v", title, err),
			menu.BackMarkup,
		)
	}

	return editOrSend(
		ctx,
		query,
		fmt.Sprintf("Чат %s успешно удален.", title),
		menu.BackMarkup,
	)
}

func (h *AdminHandler) ShowChat(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")
	if len(parts) != 3 {
		return editOrSend(ctx, query, "Не удалось определить чат", nil)
	}

	title := parts[1]

	if title == "" {
		return editOrSend(ctx, query, "Не удалось определить title чата", nil)
	}

	chat, err := h.chatService.GetByTitle(ctx.Context(), title)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return editOrSend(
				ctx,
				query,
				"Чат не найден",
				tu.InlineKeyboard([]telego.InlineKeyboardButton{menu.Back}),
			)
		}
		return editOrSend(ctx, query, "Не удалось получить информацию о чатах"+err.Error(), nil)
	}

	text := fmt.Sprintf(
		"Чат: %s\nID: %d\nТип: %s\nАктивен: %t",
		chat.Title,
		chat.ChatID,
		chat.Type,
		chat.IsActive,
	)
	return editOrSend(ctx, query, text, &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				menu.Back,
				telego.InlineKeyboardButton{
					Text:         "Удалить чат",
					CallbackData: fmt.Sprintf("delete_chat;%s;%d", chat.Title, chat.ID),
				},
			},
		},
	})
}

func (h *AdminHandler) ManageCrons(ctx *th.Context, query telego.CallbackQuery) error {
	mkp := telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{{
		menu.StartCron,
	}, {
		menu.StopCron,
	}, {
		menu.Back,
	}}}

	return editOrSend(ctx, query, "Управляйте автоматическими рассылками отчётов.", &mkp)
}

// Back button: return to admin main menu
func (h *AdminHandler) Back(ctx *th.Context, query telego.CallbackQuery) error {
	_ = h.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{CallbackQueryID: query.ID})

	rmkp := tu.InlineKeyboard(
		tu.InlineKeyboardCols(
			1,
			menu.ShowReportsAdmin,
			menu.ManageCron,
			menu.ManageChats,
			menu.ManageUsers,
		)...)

	return editOrSend(ctx, query, "Панель администратора. Выберите нужное действие.", rmkp)
}

func (h *AdminHandler) ListCrons(ctx *th.Context, query telego.CallbackQuery) error {
	return editOrSend(
		ctx,
		query,
		"Расписания можно запускать и останавливать через кнопки ниже",
		menu.BackMarkup,
	)
}

func (h *AdminHandler) SwitchCronStatus(ctx *th.Context, query telego.CallbackQuery) error {
	return editOrSend(ctx, query, "Выберите действие для расписаний", menu.BackMarkup)
}

func (h *AdminHandler) StartJobs(ctx *th.Context, query telego.CallbackQuery) error {
	h.report.Start()
	return editOrSend(
		ctx,
		query,
		"Рассылки запущены. Автоматические отчёты снова будут приходить по расписанию.",
		menu.BackMarkup,
	)
}

func (h *AdminHandler) StopJobs(ctx *th.Context, query telego.CallbackQuery) error {
	h.report.Stop()
	return editOrSend(
		ctx,
		query,
		"Рассылки остановлены. Автоматические отчёты временно не будут приходить.",
		menu.BackMarkup,
	)
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
			Text:        "Панель администратора. Выберите нужное действие.",
			ReplyMarkup: rmkp,
		},
	)

	return err
}
