package handlers

import (
	"context"
	"log/slog"

	"github.com/go-playground/validator/v10"
)

type AuthProvider interface {
	Login(ctx context.Context, email, password string) (string, string, error)
	Refresh(ctx context.Context, refreshToken string) (string, string, error)
}

type AuthHandler struct {
	validator *validator.Validate
	auth      AuthProvider
	log       *slog.Logger
}

func NewAuthHandler(auth AuthProvider, log *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, log: log, validator: validator.New()}
}

type ReportSchemeProvider interface {
	GetReportList(ctx context.Context, limit, offset uint)
}

type ReportSchemeHandler struct {
	log *slog.Logger
}

type TemplateProvider interface {
	GetReportList(ctx context.Context, limit, offset uint)
}

type TemplateHandler struct {
	log *slog.Logger
}

type CronHandler struct {
	log *slog.Logger
}

type QueryHandler struct {
	log *slog.Logger
}

type RecipientHandler struct {
	log *slog.Logger
}
type EmailTemplateHandler struct {
	log *slog.Logger
}
