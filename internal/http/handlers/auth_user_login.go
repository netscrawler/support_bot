package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	domainErr "support_bot/internal/errorz"
	"support_bot/internal/http/dto"
	"support_bot/internal/http/errorz"
	httputils "support_bot/internal/http/utils"
)

func (h *AuthHandler) AuthUserLogin(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		AnswerErrJSON(w, errorz.ErrInvalidRequest)
		return
	}

	authDTO, err := httputils.UnmarshalFor[dto.AuthUserRequestDTO](body)
	if err != nil {
		AnswerErrJSON(w, errorz.ErrInvalidRequest)
		return
	}

	validationErr := authDTO.Validate(h.validator)
	if validationErr != nil {
		AnswerErrJSON(w, errorz.ClientErr{
			Code: http.StatusBadRequest,
			Desc: validationErr.Error(),
		})
		return
	}

	tok, err := h.auth.Login(r.Context(), authDTO.Email, authDTO.Password)
	if err != nil {
		var e domainErr.AuthErr
		if errors.As(err, &e) {
			AnswerErrJSON(w, errorz.ClientErr{
				Code: http.StatusUnauthorized,
				Desc: e.Error(),
			})
			return
		}
		AnswerErrJSON(w, errorz.ClientErr{
			Code: http.StatusInternalServerError,
		})
	}

	resp := dto.NewAuthUserResponseDTOfromDomain(tok)

	err = HandleJSON(w, http.StatusOK, resp)
	if err != nil {
		h.log.WarnContext(r.Context(), "response marshall error", slog.Any("error", err))
	}
}
