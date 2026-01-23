package handlers

import (
	tele "gopkg.in/telebot.v4"
	models "support_bot/internal/models/notify"
)

type TextHandler struct {
	adminhandler *AdminHandler
	userhandler  *UserHandler
	state        *State
}

func NewTextHandler(
	adminhandler *AdminHandler,
	userhandler *UserHandler,
	state *State,
) *TextHandler {
	return &TextHandler{
		adminhandler: adminhandler,
		userhandler:  userhandler,
		state:        state,
	}
}

func (h *TextHandler) ProcessTextInput(c tele.Context) error {
	if c.Get("role") == models.UserRole {
		return h.userhandler.ProcessUserInput(c)
	}

	if c.Get("role") == models.AdminRole || c.Get("role") == models.PrimaryAdminRole {
		return h.adminhandler.ProcessAdminInput(c)
	}

	return nil
}
