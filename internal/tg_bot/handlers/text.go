package handlers

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
