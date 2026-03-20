package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	domainErr "support_bot/internal/domain/errorz"
	"support_bot/internal/http/dto"
	"support_bot/internal/http/errorz"
	httputils "support_bot/internal/http/utils"
)

func (h AuthHandler) AuthUserRefresh(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		AnswerErrJSON(w, errorz.ErrInvalidRequest)

		return
	}

	defer r.Body.Close()

	refreshDTO, err := httputils.UnmarshalFor[dto.AuthUserRefreshRequestDTO](body)
	if err != nil {
		AnswerErrJSON(w, errorz.ErrInvalidRequest)

		return
	}

	validationErr := refreshDTO.Validate(h.validator)
	if validationErr != nil {
		AnswerErrJSON(w, errorz.ClientErr{
			Code: http.StatusBadRequest,
			Desc: validationErr.Error(),
		})

		return
	}

	tok, refreshTok, err := h.auth.Refresh(r.Context(), refreshDTO.RefreshToken)
	if err != nil {
		if errors.Is(err, domainErr.ErrTokenExpired) || errors.Is(err, domainErr.ErrInvalidToken) {
			AnswerErrJSON(w, errorz.ClientErr{
				Code: http.StatusUnauthorized,
				Desc: err.Error(),
			})

			return
		}

		AnswerErrJSON(w, errorz.ClientErr{
			Code: http.StatusInternalServerError,
			Desc: err.Error(),
		})

		return
	}

	resp := dto.AuthUserRefreshResponseDTO{
		AccessToken:  tok,
		RefreshToken: refreshTok,
	}

	err = HandleJSON(w, http.StatusOK, resp)
	if err != nil {
		h.log.WarnContext(r.Context(), "response marshall error", slog.Any("error", err))
	}
}
