package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"support_bot/internal/api/http/middlewares"
	"support_bot/internal/pkg/httplib"
)

type Server struct {
	srv    *http.Server
	router *httplib.Router

	cfg *Config
	log *slog.Logger
}

func New(cfg *Config, log *slog.Logger) *Server {
	router := httplib.NewRouter()
	mw := middlewares.NewMiddleware(log, cfg.MaxBodyBytes, cfg.AuthToken)

	router.Use(
		mw.RecoverMiddleware,
		mw.Trace,
		mw.Gzip,
		mw.LogRequest,
		mw.Auth,
	)

	return &Server{
		srv: &http.Server{
			Addr:                cfg.Addr(),
			Handler:             router.Mux(),
			ReadTimeout:         cfg.ReadTimeout,
			ReadHeaderTimeout:   cfg.ReadHeaderTimeout,
			WriteTimeout:        cfg.WriteTimeout,
			IdleTimeout:         cfg.IdleTimeout,
			MaxHeaderBytes:      cfg.MaxHeaderBytes,
			MaxHeaderValueCount: cfg.MaxHeaderValueCount,
		},
		cfg:    cfg,
		router: router,
		log:    log,
	}
}

func (s *Server) Addr() string {
	return s.srv.Addr
}

// Router returns the underlying router so callers can register routes
// after construction, once all handler dependencies are built.
func (s *Server) Router() *httplib.Router {
	return s.router
}

func (s *Server) Run() error {
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Start() {
	go func() {
		if err := s.Run(); err != nil {
			s.log.Error("http server stopped", slog.Any("error", err))
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
