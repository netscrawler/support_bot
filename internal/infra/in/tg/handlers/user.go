package handlers

import (
	"context"
	"errors"
	"fmt"

	tele "gopkg.in/telebot.v4"
	"support_bot/internal/infra/in/tg/menu"
	"support_bot/internal/models"
	"support_bot/internal/service"
)

type UserHandler struct {
	bot         *tele.Bot
	chatService *service.Chat
	userService *service.User
	state       *State
	notify      *service.TelegramNotify
}

func NewUserHandler(
	bot *tele.Bot,
	chatService *service.Chat,
	userService *service.User,
	state *State,
	notify *service.TelegramNotify,
) *UserHandler {
	return &UserHandler{
		bot:         bot,
		chatService: chatService,
		userService: userService,
		state:       state,
		notify:      notify,
	}
}

func (h *UserHandler) ProcessUserInput(c tele.Context) error {
	userID := c.Sender().ID
	state := h.state.Get(userID)

	switch state {
	case SendNotificationState:
		return h.ProcessSendNotification(c)
	case ConfirmNotificationState:
		return h.ConfirmSendNotification(c)
	case CancelNotificationState:
		return h.CancelSendNotification(c)
	default:
		return nil
	}
}

func (h *UserHandler) StartUser(c tele.Context) error {
	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}

	menu.UserMenu.Reply(
		menu.UserMenu.Row(menu.SendNotifyUser),
	)
	//nolint:errcheck
	c.Delete()
	h.state.Set(c.Sender().ID, MenuState)

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
	err := h.userService.AddUserComplete(&snd)
	formatedString := fmt.Sprintf(
		"Пользователь с ником @%s успешно прошел регистрацию",
		c.Sender().Username,
	)
	//nolint:errcheck
	h.notify.SendAdminNotify(ctx, formatedString)

	if err == nil {
		return c.Send("Вы успешно прошли регистрацию!\n напишите /start чтобы начать работу")
	}

	return nil
}

func (h *UserHandler) SendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, SendNotificationState)
	//nolint:errcheck
	c.Delete()

	return c.Send(PleaseSendMessage)
}

func (h *UserHandler) ProcessSendNotification(c tele.Context) error {
	if h.state.Get(c.Sender().ID) != SendNotificationState {
		return c.Edit("Время на отправку истекло, начните заново")
	}

	msg := c.Text()
	if msg == "" {
		return c.Send(PleaseSendMessage)
	}

	h.state.SetMsgData(c.Sender().ID, msg)

	confirmBtn := menu.Selector.Data(
		"✅ Отправить",
		"confirm_user_notification",
	)
	cancelBtn := menu.Selector.Data("❌ Отменить", "cancel_user_notification")

	menu.Selector.Inline(
		menu.Selector.Row(cancelBtn, confirmBtn),
	)

	h.state.Set(c.Sender().ID, ConfirmNotificationState)

	conf := "Вы уверены, что хотите отправить это уведомление?\n\n"
	formated := fmt.Sprintf("%s```\n%s```", conf, msg)

	// Отправляем сообщение с клавиатурой
	return c.Send(
		formated,
		menu.Selector,
		tele.ModeMarkdownV2,
	)
}

func (h *UserHandler) ConfirmSendNotification(c tele.Context) error {
	ctx := context.Background()

	msg, ok := h.state.GetMsgData(c.Sender().ID)
	if h.state.Get(c.Sender().ID) != ConfirmNotificationState || !ok {
		return c.Edit("Время на подтверждение истекло")
	}

	resp, err := h.notify.BroadcastToChats(ctx, msg)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return c.Edit(UnableCauseNotFound)
		}

		if errors.Is(err, models.ErrInternal) {
			return c.Edit(UnableCauseInternal)
		}

		return c.Edit(UnableSendMessages + err.Error())
	}

	userString := fmt.Sprintf("Пользователь @%s разослал уведомление:", c.Sender().Username)
	formString := fmt.Sprintf(
		"%s\n```\n%s```",
		userString, msg,
	)
	//nolint:errcheck
	go h.notify.SendAdminNotify(ctx, formString)

	h.state.Set(c.Sender().ID, MenuState)

	return c.Edit(resp, tele.ModeMarkdownV2)
}

func (h *UserHandler) CancelSendNotification(c tele.Context) error {
	h.state.Set(c.Sender().ID, MenuState)

	return c.Edit("❌ Отправка уведомления отменена.")
}

func (h *UserHandler) UserAuthMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
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
