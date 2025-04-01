package handlers

import (
	"support_bot/internal/models"

	tele "gopkg.in/telebot.v4"
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
	if c.Get("isAdmin") == models.UserRole {
		return h.userhandler.ProcessUserInput(c)
	}
	if c.Get("isAdmin") == models.AdminRole || c.Get("isAdmin") == models.PrimaryAdminRole {
		return h.adminhandler.ProcessAdminInput(c)
	}
	return nil
}
