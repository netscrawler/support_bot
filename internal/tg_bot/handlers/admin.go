package handlers

import (
	"context"
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
		nil,
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
			ChatID:    telego.ChatID{ID: query.Message.GetChat().ID},
			MessageID: query.Message.Message().MessageID,
			Text:      "Не удалось запустить отчет",
		})
		return err
	}

	return editOrSend(
		ctx,
		query,
		"Отчёт поставлен в очередь. Результат придёт в этот чат в течение нескольких минут.",
		nil,
	)
}

func (h *AdminHandler) ManageUsers(ctx *th.Context, query telego.CallbackQuery) error {
	mkp := telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{{
		menu.ListUser,
	}, {
		menu.AddUser,
	}, {
		menu.RemoveUser,
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
		return editOrSend(ctx, query, "Не удалось получить список пользователей", nil)
	}

	if len(users) == 0 {
		return editOrSend(ctx, query, "Список пользователей пуст", nil)
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
			return editOrSend(ctx, query, "Не удалось получить список пользователей", nil)
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
			nil,
		)
	}

	return editOrSend(ctx, query, fmt.Sprintf("Пользователь %s удалён", username), nil)
}

func (h *AdminHandler) ShowUser(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")
	if len(parts) != 2 {
		return editOrSend(ctx, query, "Не удалось определить пользователя", nil)
	}

	user, err := h.userService.GetByUsername(ctx.Context(), parts[1])
	if err != nil {
		return editOrSend(ctx, query, "Не удалось получить информацию о пользователе", nil)
	}

	text := fmt.Sprintf(
		"Пользователь: %s\nID: %d\nРоль: %s",
		user.Username,
		user.TelegramID,
		user.Role,
	)
	return editOrSend(ctx, query, text, nil)
}

func (h *AdminHandler) ManageChats(ctx *th.Context, query telego.CallbackQuery) error {
	mkp := telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{{
		menu.ListChats,
	}, {
		menu.RemoveChat,
	}}}

	return editOrSend(ctx, query, "Выберите действие для управления чатами.", &mkp)
}

func (h *AdminHandler) AddChat(ctx *th.Context, query telego.CallbackQuery) error {
	h.state.set(query.Message.GetChat().ID, addChatState)
	return editOrSend(
		ctx,
		query,
		"Введите чат в формате <chat_id> <title> или просто <title>.\nНапример: 123456789 Бухгалтерия",
		nil,
	)
}

func (h *AdminHandler) RemoveChat(ctx *th.Context, query telego.CallbackQuery) error {
	h.state.set(query.Message.GetChat().ID, removeChatState)
	return editOrSend(ctx, query, "Введите точное название чата для удаления.", nil)
}

func (h *AdminHandler) ListChats(ctx *th.Context, query telego.CallbackQuery) error {
	chats, err := h.chatService.GetAll(ctx.Context())
	if err != nil {
		return editOrSend(ctx, query, "Не удалось получить список чатов", nil)
	}

	if len(chats) == 0 {
		return editOrSend(ctx, query, "Список чатов пуст", nil)
	}

	rows := make([][]telego.InlineKeyboardButton, 0, len(chats)+1)
	for _, chat := range chats {
		rows = append(rows, []telego.InlineKeyboardButton{{
			Text:         fmt.Sprintf("💬 %s", chat.Title),
			CallbackData: fmt.Sprintf("show_chat;%s", chat.Title),
		}})
	}

	rows = append(rows, []telego.InlineKeyboardButton{{
		Text:         "🔙 Назад",
		CallbackData: "manage_chats",
	}})

	mkp := telego.InlineKeyboardMarkup{InlineKeyboard: rows}
	return editOrSend(ctx, query, "Список чатов", &mkp)
}

func (h *AdminHandler) DeleteChats(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")
	if len(parts) == 1 || parts[1] == "" {
		chats, err := h.chatService.GetAll(ctx.Context())
		if err != nil {
			return editOrSend(ctx, query, "Не удалось получить список чатов", nil)
		}

		rows := make([][]telego.InlineKeyboardButton, 0, len(chats)+1)
		for _, chat := range chats {
			rows = append(rows, []telego.InlineKeyboardButton{{
				Text:         fmt.Sprintf("❌ %s", chat.Title),
				CallbackData: fmt.Sprintf("delete_chat;%s", chat.Title),
			}})
		}

		rows = append(rows, []telego.InlineKeyboardButton{{
			Text:         "🔙 Назад",
			CallbackData: "manage_chats",
		}})

		mkp := telego.InlineKeyboardMarkup{InlineKeyboard: rows}
		return editOrSend(ctx, query, "Выберите чат для удаления", &mkp)
	}

	title := parts[1]
	if err := h.chatService.Remove(ctx.Context(), title); err != nil {
		return editOrSend(ctx, query, fmt.Sprintf("Не удалось удалить чат %s", title), nil)
	}

	return editOrSend(ctx, query, fmt.Sprintf("Чат %s удалён", title), nil)
}

func (h *AdminHandler) ShowChats(ctx *th.Context, query telego.CallbackQuery) error {
	parts := strings.Split(query.Data, ";")
	if len(parts) != 2 {
		return editOrSend(ctx, query, "Не удалось определить чат", nil)
	}

	chats, err := h.chatService.GetAll(ctx.Context())
	if err != nil {
		return editOrSend(ctx, query, "Не удалось получить информацию о чатах", nil)
	}

	for _, chat := range chats {
		if chat.Title == parts[1] {
			text := fmt.Sprintf(
				"Чат: %s\nID: %d\nТип: %s\nАктивен: %t",
				chat.Title,
				chat.ChatID,
				chat.Type,
				chat.IsActive,
			)
			return editOrSend(ctx, query, text, nil)
		}
	}

	return editOrSend(ctx, query, "Чат не найден", nil)
}

func (h *AdminHandler) HandleTextMessage(ctx *th.Context, message telego.Message) error {
	state := h.state.get(message.Chat.ID)
	if state == "" {
		return nil
	}

	text := strings.TrimSpace(message.Text)
	// Allow cancellation — keep state on validation errors to allow retry
	if strings.EqualFold(text, "отмена") || strings.EqualFold(text, "/cancel") {
		h.state.delete(message.Chat.ID)
		_, _ = ctx.Bot().
			SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Действие отменено. Возвращаю вас в меню."})
		return showAdminMenu(ctx.Bot(), message)
	}

	switch state {
	case addUserState:
		return h.processUserChange(ctx, message, true)
	case removeUserState:
		return h.processUserChange(ctx, message, false)
	case addChatState:
		return h.processChatChange(ctx, message, true)
	case removeChatState:
		return h.processChatChange(ctx, message, false)
	default:
		return nil
	}
}

// unified user add/remove
func (h *AdminHandler) processUserChange(ctx *th.Context, message telego.Message, add bool) error {
	username := strings.TrimSpace(message.Text)
	if username == "" {
		_, _ = ctx.Bot().
			SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Имя пользователя не может быть пустым. Попробуйте снова или отправьте 'отмена' чтобы прервать."})
		return nil
	}

	if add {
		if err := h.userService.CreateEmpty(ctx.Context(), username, false); err != nil {
			_, _ = ctx.Bot().
				SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Не удалось добавить пользователя. Попробуйте ещё раз."})
			return err
		}
		_, _ = ctx.Bot().
			SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Пользователь добавлен."})
	} else {
		if err := h.userService.Delete(ctx.Context(), username, false); err != nil {
			_, _ = ctx.Bot().
				SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Не удалось удалить пользователя. Убедитесь, что имя указано верно."})
			return err
		}
		_, _ = ctx.Bot().
			SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Пользователь удалён."})
	}

	h.state.delete(message.Chat.ID)
	// return to admin menu
	return showAdminMenu(ctx.Bot(), message)
}

// unified chat add/remove
func (h *AdminHandler) processChatChange(ctx *th.Context, message telego.Message, add bool) error {
	parts := strings.Fields(message.Text)
	if len(parts) == 0 {
		_, _ = ctx.Bot().
			SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Не удалось определить чат. Введите в формате: <chat_id> <title> или просто <title>."})
		return nil
	}

	chatID := int64(0)
	title := strings.TrimSpace(message.Text)
	if len(parts) >= 2 {
		if parsedID, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			chatID = parsedID
			title = strings.Join(parts[1:], " ")
		}
	}

	chat := &models.TgChatDTO{ChatID: chatID, Title: title, Type: "private", IsActive: true}
	if add {
		if err := h.chatService.Add(ctx.Context(), chat); err != nil {
			_, _ = ctx.Bot().
				SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Не удалось добавить чат. Проверьте формат и попробуйте снова."})
			return err
		}
		_, _ = ctx.Bot().
			SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Чат добавлен."})
	} else {
		if err := h.chatService.Remove(ctx.Context(), title); err != nil {
			_, _ = ctx.Bot().
				SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Не удалось удалить чат. Убедитесь, что название указано точно."})
			return err
		}
		_, _ = ctx.Bot().
			SendMessage(ctx, &telego.SendMessageParams{ChatID: telego.ChatID{ID: message.Chat.ID}, Text: "Чат удалён."})
	}

	h.state.delete(message.Chat.ID)
	// return to admin menu
	return showAdminMenu(ctx.Bot(), message)
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
		nil,
	)
}

func (h *AdminHandler) SwitchCronStatus(ctx *th.Context, query telego.CallbackQuery) error {
	return editOrSend(ctx, query, "Выберите действие для расписаний", nil)
}

func (h *AdminHandler) StartJobs(ctx *th.Context, query telego.CallbackQuery) error {
	h.report.Start()
	return editOrSend(
		ctx,
		query,
		"Рассылки запущены. Автоматические отчёты снова будут приходить по расписанию.",
		nil,
	)
}

func (h *AdminHandler) StopJobs(ctx *th.Context, query telego.CallbackQuery) error {
	h.report.Stop()
	return editOrSend(
		ctx,
		query,
		"Рассылки остановлены. Автоматические отчёты временно не будут приходить.",
		nil,
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
