package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"support_bot/internal/errorz"
	reportModels "support_bot/internal/models/report"
	"support_bot/internal/pkg"
	"support_bot/internal/tg_bot/menu"
	"support_bot/internal/tg_bot/service"
	"time"

	tele "gopkg.in/telebot.v4"

	models "support_bot/internal/models/notify"
)

type AdminHandler struct {
	bot         *tele.Bot
	userService *service.User
	chatService *service.Chat
	report      *service.Report
	state       *State
}

func NewAdminHandler(
	bot *tele.Bot,
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

// StartAdmin handles the start command for admins.
func (h *AdminHandler) StartAdmin(c tele.Context) error {
	if c.Chat().Type != tele.ChatPrivate {
		return nil
	}

	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.ManageUsers, menu.ManageChats),
		menu.AdminMenu.Row(menu.LoadAndShowReportUser, menu.ManageCron),
	)
	h.state.Set(c.Sender().ID, MenuState)
	//nolint:errcheck
	c.Delete()

	return c.Send(HelloAdminRegistration, menu.AdminMenu)
}

func (h *AdminHandler) LoadReports(c tele.Context) error {
	h.state.Set(c.Sender().ID, LoadReportState)
	//nolint:errcheck
	//c.Delete()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rpl, err := h.report.LoadReportsWithPagination(ctx)
	if err != nil {
		return c.Send("Ошибка получения отчетов: " + err.Error())
	}

	mark := mapReportRPLToMarkup(rpl)

	return c.Send(menu.MsgHelloReport, &mark)
}

func (h *AdminHandler) LoadReportsPage(c tele.Context) error {
	userID := c.Sender().ID
	if h.state.Get(userID) != LoadReportState {
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
	h.state.Set(userID, LoadReportState)

	if err := c.Edit(menu.MsgHelloReport, &mark); err != nil {
		if isMessageNotModified(err) {
			return nil
		}

		return err
	}

	return nil
}

func (h *AdminHandler) IgnoreReportPage(c tele.Context) error {
	return c.Respond()
}

func (h *AdminHandler) GenerateSelectedReport(c tele.Context) error {
	userID := c.Sender().ID
	if h.state.Get(userID) != LoadReportState {
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

	chat := &reportModels.Chat{
		ChatID:   c.Chat().ID,
		Title:    &c.Chat().FirstName,
		Type:     string(c.Chat().Type),
		IsActive: true,
	}

	if err := h.report.GenerateReportByName(ctx, reportName, chat); err != nil {
		return c.Edit("Не удалось запустить отчет: " + err.Error())
	}

	h.state.Set(userID, MenuState)

	if err := c.Edit("Отчет запущен. Результат придет в этот чат."); err != nil {
		if errors.Is(tele.ErrMessageNotModified, err) {
			return nil
		}

		return err
	}

	return nil
}

// ManageUsers handles the user management menu.
func (h *AdminHandler) ManageUsers(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.AddUser, menu.RemoveUser),
		menu.AdminMenu.Row(menu.ListUser, menu.Back))
	h.state.Set(c.Sender().ID, MenuState)

	c.Delete()

	return c.Send(ManageUsers, menu.AdminMenu)
}

func (h *AdminHandler) ProcessAdminInput(c tele.Context) error {
	userID := c.Sender().ID

	if h.state.Get(userID) == MenuState {
		return nil
	}

	state := h.state.Get(userID)

	switch state {
	case AddUserState:
		return h.ProcessAddUser(c) // Вызываем обработку добавления пользователя
	case RemoveUserState:
		return h.ProcessRemoveUser(c)
	case AddChatState:
		return h.ProcessAddChat(c)
	case RemoveChatState:
		return h.ProcessRemoveChat(c)
	default:
		return nil // Если нет активного состояния — игнорируем
	}
}

func (h *AdminHandler) AddUser(c tele.Context) error {
	h.state.Set(c.Sender().ID, AddUserState)

	c.Delete()

	return c.Send(
		UserAddRemove,
	)
}

// ProcessAddUser processes the username input for adding a user.
func (h *AdminHandler) ProcessAddUser(c tele.Context) error {
	userID := c.Sender().ID
	if h.state.Get(userID) != AddUserState {
		return nil
	}

	c.Delete()

	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send(PleaseSendCorrectUsername)
	}

	username = username[1:]
	confirmBtn := menu.Selector.Data(
		"Admin",
		"add_admin",
		username,
	)
	cancelBtn := menu.Selector.Data("User", "add_user", username)

	menu.Selector.Inline(
		menu.Selector.Row(cancelBtn, confirmBtn),
	)
	//nolint: errcheck
	c.Delete()

	return c.Send("Выберите роль для пользователя @"+username+".", menu.Selector)
}

func (h *AdminHandler) AddUserWithUserRole(c tele.Context) error {
	ctx := context.Background()

	userID := c.Sender().ID
	if h.state.Get(userID) != AddUserState {
		return nil
	}

	username := c.Data()

	err := h.userService.CreateEmpty(ctx, username, false)
	if err != nil {
		return c.Send("Не удалось добавить пользователя: " + err.Error())
	}

	h.state.Set(userID, MenuState) // Сбрасываем состояние

	return c.Edit("Пользователь @" + username + " добавлен.")
}

func (h *AdminHandler) AddUserWithAdminRole(c tele.Context) error {
	ctx := context.Background()

	userID := c.Sender().ID
	if h.state.Get(userID) != AddUserState {
		return nil
	}

	username := c.Data()

	err := h.userService.CreateEmpty(ctx, username, true)
	if err != nil {
		return c.Edit("Не удалось добавить пользователя: " + err.Error())
	}

	h.state.Set(userID, MenuState) // Сбрасываем состояние

	return c.Edit("Администратор @" + username + " добавлен.")
}

// RemoveUser handles removing a user.
func (h *AdminHandler) RemoveUser(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveUserState)
	//nolint:errcheck
	c.Delete()

	return c.Send(
		UserAddRemove,
	)
}

// ProcessRemoveUser processes the username input for removing a user.
func (h *AdminHandler) ProcessRemoveUser(c tele.Context) error {
	ctx := context.Background()
	isPrimeReq := false

	username := c.Text()
	if !strings.HasPrefix(username, "@") {
		return c.Send(PleaseSendCorrectUsername)
	}

	// Remove @ and extract the username
	username = username[1:]
	if username == c.Sender().Username {
		return c.Send(ErrDeleteUserCauseSuicide)
	}

	role, err := h.userService.IsAllowed(ctx, c.Sender().ID)
	if role == models.PrimaryAdminRole {
		isPrimeReq = true
	}

	if err != nil {
		return c.Send(ErrDeleteUser + err.Error())
	}

	// Call service to remove user
	err = h.userService.Delete(ctx, username, isPrimeReq)
	if err != nil {
		return c.Send(ErrDeleteUser + err.Error())
	}

	h.state.Set(c.Sender().ID, MenuState)

	return c.Send("Пользователь @" + username + " успешно удален!")
}

// ListUsers handles listing all users.
func (h *AdminHandler) ListUsers(c tele.Context) error {
	ctx := context.Background()

	// c.Delete()

	users, err := h.userService.GetAll(ctx)
	if errors.Is(err, errorz.ErrNotFound) {
		return c.Send("Пользователи не найдены.")
	}

	if err != nil {
		return c.Send("Ошибка получения пользователей: " + err.Error())
	}

	var response strings.Builder

	response.WriteString("📋 *Список пользователей:*\n\n")

	for i, user := range users {
		fmt.Fprintf(&response, "%d. @%s - Role: %s\n", i+1, user.Username, user.Role)
	}

	return c.Send(
		pkg.EscapeMarkdownV2(response.String()),
		&tele.SendOptions{ParseMode: tele.ModeMarkdownV2},
	)
}

// ManageChats handles the chat management menu.
func (h *AdminHandler) ManageChats(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.RemoveChat),
		menu.AdminMenu.Row(menu.ListChats, menu.Back))
	//nolint:errcheck
	c.Delete()

	return c.Send("Управление чатами", menu.AdminMenu)
}

// ProcessAddActiveChat processes the chat input for adding a chat.
func (h *AdminHandler) ProcessAddActiveChat(c tele.Context) error {
	ctx := context.Background()

	if c.Chat().Type == tele.ChatPrivate {
		return c.Send("Эта команда может использоваться только в чатах")
	}

	c.Delete()

	chatToAdd := models.NewChat(
		c.Chat().ID,
		c.Chat().Title,
		string(c.Chat().Type),
		c.Chat().Description,
	)
	chatToAdd.Activate()

	_ = h.chatService.AddActive(ctx, chatToAdd)
	//if err != nil {
	//	h.notify.SendNotify(
	//		ctx,
	//		c.Sender().MessageID,
	//		fmt.Sprintf("Ошибка добавления чата: %s : %v", c.Chat().Title, err.Error()),
	//	)
	//
	//	return nil
	//}
	//
	//h.notify.BroadcastToUsers(
	//	ctx,
	//	"Добавлен новый чат в рассылку: "+c.Chat().Title,
	//)

	return nil
}

// ProcessAddChat processes the chat input for adding a chat.
func (h *AdminHandler) ProcessAddChat(c tele.Context) error {
	ctx := context.Background()

	if c.Chat().Type == tele.ChatPrivate {
		return c.Send("Эта команда может использоваться только в чатах")
	}

	c.Delete()

	chatToSave := models.NewChat(
		c.Chat().ID,
		c.Chat().Title,
		string(c.Chat().Type),
		c.Chat().Description,
	)

	_ = h.chatService.Add(ctx, chatToSave)
	//if err != nil {
	//	h.notify.SendNotify(
	//		ctx,
	//		c.Sender().MessageID,
	//		fmt.Sprintf("Ошибка добавления чата: %s : %v", c.Chat().Title, err.Error()),
	//	)
	//
	//	return nil
	//}
	//
	//h.notify.BroadcastToUsers(
	//	ctx,
	//	"Добавлен новый чат: "+c.Chat().Title,
	//)

	return nil
}

// ProcessInfoCommand processes the chat input for adding a chat.
func (h *AdminHandler) ProcessInfoCommand(c tele.Context) error {
	if c.Chat().Type == tele.ChatPrivate {
		return c.Send("Эта команда может использоваться только в чатах")
	}

	c.Delete()

	ans := fmt.Sprintf(
		"*Информация о чате:*\n Title: `%s`\n MessageID: `%d`\n Thread: `%d`",
		c.Chat().Title,
		c.Chat().ID,
		c.ThreadID(),
	)

	return c.Send(ans, &tele.SendOptions{ParseMode: tele.ModeMarkdownV2})
}

// RemoveChat handles removing a chat.
func (h *AdminHandler) RemoveChat(c tele.Context) error {
	h.state.Set(c.Sender().ID, RemoveChatState)
	c.Delete()

	return c.Send("Пожалуйста пришлите имя чата который вы хотите удалить.")
}

// ProcessRemoveChat processes the chat input for removing a chat.
func (h *AdminHandler) ProcessRemoveChat(c tele.Context) error {
	ctx := context.Background()

	chatName := c.Text()

	err := h.chatService.Remove(ctx, chatName)
	if err != nil {
		return c.Send("Ошибка удаления чата: " + err.Error())
	}

	h.state.Set(c.Sender().ID, MenuState)

	return c.Send("Чат " + chatName + " успешно удален!")
}

// ListChats handles listing all chats.
func (h *AdminHandler) ListChats(c tele.Context) error {
	ctx := context.Background()

	chats, err := h.chatService.GetAll(ctx)
	if err != nil {
		return c.Send("Ошибка получения чатов: " + err.Error())
	}

	if len(chats) == 0 {
		return c.Send("Чатов не найдено.")
	}

	var response strings.Builder

	response.WriteString("*Список чатов:*\n\n")

	for i, chat := range chats {
		fmt.Fprintf(&response, "%d. %s\n", i+1, chat.Title)
	}

	s := pkg.EscapeMarkdownV2(response.String())

	return c.Send(
		s,
		&tele.SendOptions{ParseMode: tele.ModeMarkdownV2},
	)
}

// ManageCron handles the chat management menu.
func (h *AdminHandler) ManageCron(c tele.Context) error {
	menu.AdminMenu.Reply(
		menu.AdminMenu.Row(menu.StartCron),
		menu.AdminMenu.Row(menu.StopCron, menu.Back))

	c.Delete()

	return c.Send("Управление задачами", menu.AdminMenu)
}

// StartCronJobs перезапускает крон-задачи для уведомлений.
func (h *AdminHandler) StartCronJobs(c tele.Context) error {
	ctx := context.Background()

	ans := h.startJobs(ctx)

	return c.Send(ans)
}

// StopCronJobs перезапускает крон-задачи для уведомлений.
func (h *AdminHandler) StopCronJobs(c tele.Context) error {
	h.report.Stop()

	return c.Send("Задачи успешно остановлены")
}

func (h *AdminHandler) startJobs(ctx context.Context) string {
	h.report.Start()

	return "Задачи запущены"
}
