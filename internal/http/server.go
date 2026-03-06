package http

import (
	"support_bot/internal/http/handlers"
)

type Server struct {
	handlers.AuthHandler
	handlers.ReportSchemeHandler
	handlers.CronHandler
	handlers.TemplateHandler
	handlers.QueryHandler
	handlers.RecipientHandler
	handlers.EmailTemplateHandler
}

func (r *Server) Setup() {
	HanlePOSTFunc("/user/auth", r.AuthUserLogin)

	HanleGETFunc("/report_scheme/list", r.GetReportList)
	HanleGETFunc("/report_scheme/{id}", r.GetReportByID)
	HanlePOSTFunc("/report_scheme", r.GetReportByID)
	HanlePUTFunc("/report_scheme/{id}", r.GetReportByID)
	HanleDELETEFunc("/report_scheme/{id}", r.GetReportByID)

	HanleGETFunc("/template/list", r.GetTemplateList)
	HanleGETFunc("/template/{id}", r.GetTemplateByID)
	HanlePOSTFunc("/template", r.GetReportByID)
	HanlePUTFunc("/template/{id}", r.GetReportByID)
	HanleDELETEFunc("/template/{id}", r.GetReportByID)

	HanleGETFunc("/cron/list", r.GetCronList)
	HanleGETFunc("/cron/{id}", r.GetCronByID)
	HanlePOSTFunc("/cron", r.GetReportByID)
	HanlePUTFunc("/cron/{id}", r.GetCronByID)
	HanleDELETEFunc("/cron/{id}", r.GetCronByID)

	HanleGETFunc("/query/list", r.GetQueryList)
	HanleGETFunc("/query/{id}", r.GetQueryByID)
	HanlePOSTFunc("/query", r.GetReportByID)
	HanlePUTFunc("/query/{id}", r.GetQueryByID)
	HanleDELETEFunc("/query/{id}", r.GetQueryByID)

	HanleGETFunc("/recipient/list", r.GetRecipientList)
	HanleGETFunc("/recipient/{id}", r.GetRecipientByID)
	HanlePOSTFunc("/recipient", r.GetRecipientByID)
	HanlePUTFunc("/recipient/{id}", r.GetRecipientByID)
	HanleDELETEFunc("/recipient/{id}", r.GetRecipientByID)

	HanleGETFunc("/email_template/list", r.GetEmailTemplateList)
	HanleGETFunc("/email_template/{id}", r.GetEmailTemplateByID)
	HanlePOSTFunc("/email_template", r.GetRecipientByID)
	HanlePUTFunc("/email_template/{id}", r.GetEmailTemplateByID)
	HanleDELETEFunc("/email_template/{id}", r.GetEmailTemplateByID)
}
