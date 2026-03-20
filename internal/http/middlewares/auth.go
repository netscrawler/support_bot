package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"support_bot/internal/domain/errorz"
	"support_bot/internal/pkg/logger"
)

type accessChecker interface {
	CheckAdminAccess(ctx context.Context, token string) (string, bool, error)

	CheckUserAccess(ctx context.Context, token string) (string, bool, error)
}

type AuthMiddleware struct {
	checker accessChecker
}

func NewAuthMiddleware(checker accessChecker) *AuthMiddleware {
	return &AuthMiddleware{
		checker: checker,
	}
}

func (m *AuthMiddleware) UserHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessHandler(w, r, next, m.checker.CheckUserAccess)
	})
}

func (m *AuthMiddleware) AdminHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessHandler(w, r, next, m.checker.CheckAdminAccess)
	})
}

func accessHandler(w http.ResponseWriter, r *http.Request, next http.Handler, checkFn func(context.Context, string) (string, bool, error)) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(auth, "Bearer ")

	id, ok, err := checkFn(r.Context(), token)
	if err != nil {
		if errors.Is(err, errorz.ErrInternalServer) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)

		resp, mErr := json.Marshal(map[string]string{"error": err.Error()})
		if mErr != nil {
			return
		}

		w.Write(resp)

		return
	}

	if !ok {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	ctx := logger.AppendCtx(r.Context(), slog.Any("user_id", id))
	ctx = context.WithValue(ctx, "user_id", id)
	next.ServeHTTP(w, r.WithContext(ctx))
}
