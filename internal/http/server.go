package http

import (
	"log/slog"
	"net/http"
	"support_bot/internal/http/handlers"
	"support_bot/internal/http/middlewares"
)

type Server struct {
	*handlers.AuthHandler
	// handlers.ReportSchemeHandler
	// handlers.CronHandler
	// handlers.TemplateHandler
	// handlers.QueryHandler
	// handlers.RecipientHandler
	// handlers.EmailTemplateHandler

	log  *middlewares.LogMW
	auth *middlewares.AuthMiddleware
	cors *middlewares.CORSMiddleware
	mux  *http.ServeMux
}

func NewServer(auth *handlers.AuthHandler, cors *middlewares.CORSMiddleware, authMw *middlewares.AuthMiddleware, mux *http.ServeMux, log *slog.Logger) *Server {
	return &Server{AuthHandler: auth, log: middlewares.NewLogMW(log), mux: mux, auth: authMw, cors: cors}
}

func (s *Server) Setup() {
	router := NewRouter(s.mux)
	router.Use(
		middlewares.TraceMW,
		s.cors.Handler,
		s.log.ServeHTTPFunc,
		middlewares.NewRecoveryMiddleware(slog.Default()),
	)

	auth := router.Group("/auth")
	auth.POST("/login", s.AuthUserLogin)
	auth.POST("/refresh", s.AuthUserRefresh)

	admin := router.Group("/admin")
	admin.Use(s.auth.AdminHandler)
	admin.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
		slog.Default().DebugContext(r.Context(), "test")
	})

	user := router.Group("/user")
	user.Use(s.auth.UserHandler)

	// HanleGETFunc("/report_scheme/list", s.GetReportList)
	// HanleGETFunc("/report_scheme/{id}", s.GetSchemeByID)
	// HanlePOSTFunc("/report_scheme", s.GetReportByID)
	// HanlePUTFunc("/report_scheme/{id}", s.GetReportByID)
	// HanleDELETEFunc("/report_scheme/{id}", s.GetReportByID)

	//HanleGETFunc("/template/list", s.GetTemplateList)
	//HanleGETFunc("/template/{id}", s.GetTemplateByID)
	//HanlePOSTFunc("/template", s.GetReportByID)
	//HanlePUTFunc("/template/{id}", s.GetReportByID)
	//HanleDELETEFunc("/template/{id}", s.GetReportByID)
	//
	//HanleGETFunc("/cron/list", s.GetCronList)
	//HanleGETFunc("/cron/{id}", s.GetCronByID)
	//HanlePOSTFunc("/cron", s.GetReportByID)
	//HanlePUTFunc("/cron/{id}", s.GetCronByID)
	//HanleDELETEFunc("/cron/{id}", s.GetCronByID)
	//
	//HanleGETFunc("/query/list", s.GetQueryList)
	//HanleGETFunc("/query/{id}", s.GetQueryByID)
	//HanlePOSTFunc("/query", s.GetReportByID)
	//HanlePUTFunc("/query/{id}", s.GetQueryByID)
	//HanleDELETEFunc("/query/{id}", s.GetQueryByID)
	//
	//HanleGETFunc("/recipient/list", s.GetRecipientList)
	//HanleGETFunc("/recipient/{id}", s.GetRecipientByID)
	//HanlePOSTFunc("/recipient", s.GetRecipientByID)
	//HanlePUTFunc("/recipient/{id}", s.GetRecipientByID)
	//HanleDELETEFunc("/recipient/{id}", s.GetRecipientByID)
	//
	//HanleGETFunc("/email_template/list", s.GetEmailTemplateList)
	//HanleGETFunc("/email_template/{id}", s.GetEmailTemplateByID)
	//HanlePOSTFunc("/email_template", s.GetRecipientByID)
	//HanlePUTFunc("/email_template/{id}", s.GetEmailTemplateByID)
	//HanleDELETEFunc("/email_template/{id}", s.GetEmailTemplateByID)
}

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe("localhost:8080", s.mux)
}

func (s *Server) Shutdown() {
}
