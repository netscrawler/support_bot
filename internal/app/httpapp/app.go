package httpapp

import (
	"log/slog"
	"net/http"
	"support_bot/internal/domain/tokens"
	"support_bot/internal/domain/user"
	httpSrv "support_bot/internal/http"
	"support_bot/internal/http/handlers"
	"support_bot/internal/http/middlewares"
	"support_bot/internal/use_case"

	"github.com/jmoiron/sqlx"
)

type App struct {
	srv *httpSrv.Server
}

func NewApp(cfg httpSrv.HTTPConfig, db *sqlx.DB, log *slog.Logger) *App {
	log = slog.With("module", "httpapp")
	corsMw := middlewares.NewCORSMiddleware(cfg.CORS)

	jwt := tokens.NewJWTService(cfg.JWT)
	tokenRepo := tokens.NewTokenRepository(db)
	tokenSrv := tokens.NewTokenService(tokenRepo, jwt)
	userRepo := user.NewUserRepository(db)
	userSrv := user.NewUserService(userRepo)

	authUseCase := use_case.NewAuth(tokenSrv, userSrv)
	authH := handlers.NewAuthHandler(authUseCase, log)

	authMw := middlewares.NewAuthMiddleware(authUseCase)
	mux := http.DefaultServeMux

	srv := httpSrv.NewServer(authH, corsMw, authMw, mux, log)

	srv.Setup()

	return &App{
		srv: srv,
	}
}

func (a *App) Start() {
	go func() {
		_ = a.srv.ListenAndServe()
	}()
}
