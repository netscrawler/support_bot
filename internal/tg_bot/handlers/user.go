package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"
	"support_bot/internal/models"
	"support_bot/internal/tg_bot/menu"
	"support_bot/internal/tg_bot/service"
)

type UserHandler struct {
	bot         *tele.Bot
	chatService *service.Chat
	userService *service.User
	report      *service.Report
	state       *State
}

func NewUserHandler(
	bot *tele.Bot,
	chatService *service.Chat,
	userService *service.User,
	reportService *service.Report,
	state *State,
) *UserHandler {
	return &UserHandler{
		bot:         bot,
		chatService: chatService,
		userService: userService,
		report:      reportService,
		state:       state,
	}
}

func (h *UserHandler) processUserInput(c tele.Context) error {
	userID := c.Sender().ID
	state := h.state.get(userID)

	switch state {
	case menuState:
		return h.LoadReports(c)
	// case SendNotificationState:
	//	return h.ProcessSendNotification(c)
	// case ConfirmNotificationState:
	//	return h.ConfirmSendNotification(c)
	// case CancelNotificationState:
	//	return h.CancelSendNotification(c)
	default:
		return nil
	}
}

func (h *UserHandler) StartUser(c tele.Context) error {
	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}

	menu.UserMenu.Reply(
		menu.UserMenu.Row(menu.LoadAndShowReportUser),
	)
	//nolint:errcheck
	c.Delete()
	h.state.set(c.Sender().ID, menuState)

	return c.Send("Добро пожаловать!", menu.UserMenu)
}

func (h *UserHandler) RegisterUser(c tele.Context) error {
	//nolint:errcheck
	c.Delete()

	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}

	ctx := context.Background()
	snd := models.NewUser(
		c.Sender().ID,
		c.Sender().Username,
		c.Sender().FirstName,
		&c.Sender().LastName,
		false,
	)
	err := h.userService.AddUserComplete(ctx, &snd)
	// formatedString := fmt.Sprintf(
	//	"Пользователь с ником @%s успешно прошел регистрацию",
	//	c.Sender().Username,
	//)
	//nolint:errcheck
	// h.notify.SendAdminNotify(ctx, formatedString)

	if err == nil {
		return c.Send("Вы успешно прошли регистрацию!\n напишите /start чтобы начать работу")
	}

	return nil
}

func (h *UserHandler) LoadReports(c tele.Context) error {
	h.state.set(c.Sender().ID, loadReportState)
	//nolint:errcheck
	// c.Delete()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rpl, err := h.report.LoadReportsWithPagination(ctx)
	if err != nil {
		return c.Send("Ошибка получения отчетов: " + err.Error())
	}

	mark := mapReportRPLToMarkup(rpl)

	return c.Send(menu.MsgHelloReport, &mark)
}

func (h *UserHandler) LoadReportsPage(c tele.Context) error {
	userID := c.Sender().ID
	if h.state.get(userID) != loadReportState {
		return c.Edit("Время на выбор отчета истекло, начните заново")
	}

	page, err := strconv.Atoi(c.Data())
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Не удалось определить страницу"})
	}

	if err := c.Respond(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rpl, err := h.report.LoadReportByPage(ctx, page)
	if err != nil {
		return c.Edit("Ошибка получения отчетов: " + err.Error())
	}

	mark := mapReportRPLToMarkup(rpl)

	h.state.set(userID, loadReportState)

	if err := c.Edit(menu.MsgHelloReport, &mark); err != nil {
		if isMessageNotModified(err) {
			return nil
		}

		return err
	}

	return nil
}

func (h *UserHandler) IgnoreReportPage(c tele.Context) error {
	return c.Respond()
}

func (h *UserHandler) GenerateSelectedReport(c tele.Context) error {
	userID := c.Sender().ID
	if h.state.get(userID) != loadReportState {
		return c.Edit("Время на выбор отчета истекло, начните заново")
	}

	_, reportName, ok := strings.Cut(c.Data(), ";")
	if !ok || reportName == "" {
		return c.Respond(&tele.CallbackResponse{Text: "Не удалось определить отчет"})
	}

	if err := c.Respond(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	chat := &models.Chat{
		ChatID:   c.Chat().ID,
		Title:    &c.Chat().FirstName,
		Type:     string(c.Chat().Type),
		IsActive: true,
	}

	if err := h.report.GenerateReportByName(ctx, reportName, chat); err != nil {
		return c.Edit("Не удалось запустить отчет: " + err.Error())
	}

	h.state.set(userID, menuState)

	if err := c.Edit("Отчет запущен. Результат придет в этот чат."); err != nil {
		if errors.Is(err, tele.ErrMessageNotModified) {
			return nil
		}

		return err
	}

	return nil
}

func isMessageNotModified(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "message is not modified")
}

func (h *UserHandler) processSendNotification(c tele.Context) error {
	if h.state.get(c.Sender().ID) != sendNotificationState {
		return c.Edit("Время на отправку истекло, начните заново")
	}

	msg := c.Text()
	if msg == "" {
		return c.Send(pleaseSendMessage)
	}

	h.state.setMsgData(c.Sender().ID, msg)

	confirmBtn := menu.Selector.Data(
		"✅ Отправить",
		"confirm_user_notification",
	)
	cancelBtn := menu.Selector.Data("❌ Отменить", "cancel_user_notification")
	menu.Selector.Inline(
		menu.Selector.Row(cancelBtn, confirmBtn),
	)

	h.state.set(c.Sender().ID, confirmNotificationState)

	conf := "Вы уверены, что хотите отправить это уведомление?\n\n"
	formated := fmt.Sprintf("%s```\n%s```", conf, msg)

	// Отправляем сообщение с клавиатурой
	return c.Send(
		formated,
		menu.Selector,
		tele.ModeMarkdownV2,
	)
}

// func (h *UserHandler) ConfirmSendNotification(c tele.Context) error {
//	ctx := context.Background()
//
//	msg, ok := h.state.GetMsgData(c.Sender().MessageID)
//	if h.state.Get(c.Sender().MessageID) != ConfirmNotificationState || !ok {
//		return c.Edit("Время на подтверждение истекло")
//	}
//
//	resp, err := h.notify.BroadcastToChats(ctx, msg)
//	if err != nil {
//		if errors.Is(err, errorz.ErrNotFound) {
//			return c.Edit(UnableCauseNotFound)
//		}
//
//		if errors.Is(err, errorz.ErrInternal) {
//			return c.Edit(UnableCauseInternal)
//		}
//
//		return c.Edit(UnableSendMessages + err.Error())
//	}
//
//	userString := fmt.Sprintf("Пользователь @%s разослал уведомление:", c.Sender().Username)
//	formString := fmt.Sprintf(
//		"%s\n<pre><code>%s</code></pre>",
//		userString, msg,
//	)
//	//nolint:errcheck
//	go h.notify.SendAdminNotify(ctx, formString)
//
//	h.state.Set(c.Sender().MessageID, MenuState)
//
//	return c.Edit(resp, tele.ModeMarkdownV2)
//}
//
// func (h *UserHandler) CancelSendNotification(c tele.Context) error {
//	h.state.Set(c.Sender().MessageID, MenuState)
//
//	return c.Edit("❌ Отправка уведомления отменена.")
//}

func (h *UserHandler) userAuthMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		ctx := context.Background()
		// Получаем username пользователя
		username := c.Sender().Username
		if username == "" {
			return nil
		}

		// Проверяем пользователя в базе
		user, err := h.userService.GetByUsername(ctx, username)
		//nolint:nilerr
		if err != nil {
			return nil
		}

		// Сохраняем пользователя в context
		c.Set("user", user)

		return next(c)
	}
}
