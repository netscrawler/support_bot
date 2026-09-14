package service

import (
	"context"
	"support_bot/internal/models"
	"testing"
)

func baseReport() models.Report {
	return models.Report{
		Name:       "r1",
		Title:      "t1",
		Evaluation: "true",
	}
}

func strPtr(s string) *string { return &s }

func TestValidateRecipient(t *testing.T) {
	v := NewReportValidation()
	ctx := context.Background()

	cases := []struct {
		name      string
		recipient models.Recipient
		wantErr   bool
	}{
		{"tg ok", models.Recipient{Name: "a", Type: models.TelegramRecipient, Chat: &models.Chat{ChType: models.ChatTypeTg}}, false},
		{"tg missing chat", models.Recipient{Name: "a", Type: models.TelegramRecipient}, true},
		{"tg wrong chat type", models.Recipient{Name: "a", Type: models.TelegramRecipient, Chat: &models.Chat{ChType: models.ChatTypeMax}}, true},
		{"max ok", models.Recipient{Name: "a", Type: models.MaxRecipient, Chat: &models.Chat{ChType: models.ChatTypeMax}}, false},
		{"email ok", models.Recipient{Name: "a", Type: models.EmailRecipient, Email: &models.EmailTemplate{Dest: []string{"x@y.z"}}}, false},
		{"email missing", models.Recipient{Name: "a", Type: models.EmailRecipient}, true},
		{"email no dest", models.Recipient{Name: "a", Type: models.EmailRecipient, Email: &models.EmailTemplate{}}, true},
		{"smb ok", models.Recipient{Name: "a", Type: models.SambaRecipient, RemotePath: strPtr("/x")}, false},
		{"smb missing path", models.Recipient{Name: "a", Type: models.SambaRecipient}, true},
		{"unsupported type", models.Recipient{Name: "a", Type: "carrier_pigeon"}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			report := baseReport()
			report.Recipients = []models.Recipient{c.recipient}

			err := v.Validate(ctx, report)
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestValidateExport(t *testing.T) {
	v := NewReportValidation()
	ctx := context.Background()

	cases := []struct {
		name    string
		export  models.Export
		wantErr bool
	}{
		{"csv ok", models.Export{Format: models.ReportFormatCsv, FileName: strPtr("f")}, false},
		{"csv missing filename", models.Export{Format: models.ReportFormatCsv}, true},
		{"html ok", models.Export{Format: models.ReportFormatHTML, FileName: strPtr("f"), Template: &models.Template{TemplateText: "<p>x</p>"}}, false},
		{"html missing template", models.Export{Format: models.ReportFormatHTML, FileName: strPtr("f")}, true},
		{"html empty template text", models.Export{Format: models.ReportFormatHTML, FileName: strPtr("f"), Template: &models.Template{}}, true},
		{"text ok without filename", models.Export{Format: models.ReportFormatText, Template: &models.Template{TemplateText: "x"}}, false},
		{"unsupported format", models.Export{Format: "carrier_pigeon"}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			report := baseReport()
			report.Exports = []models.Export{c.export}

			err := v.Validate(ctx, report)
			if (err != nil) != c.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}
