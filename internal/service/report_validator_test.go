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

//go:fix inline
func strPtr(s string) *string { return new(s) }

// TestValidateRecipient проверяет валидацию получателя отчета: для каждого
// типа получателя (Telegram, Max, email, SMB) на входе подаются корректные и
// заведомо неполные данные, ожидается наличие/отсутствие ошибки валидации.
func TestValidateRecipient(t *testing.T) {
	v := NewReportValidation()
	ctx := context.Background()

	cases := []struct {
		name      string
		recipient models.Recipient
		wantErr   bool
	}{
		{
			"tg: корректный чат — ошибки нет",
			models.Recipient{
				Name: "a",
				Type: models.TelegramRecipient,
				Chat: &models.Chat{ChType: models.ChatTypeTg},
			},
			false,
		},
		{
			"tg: чат не указан — ошибка",
			models.Recipient{Name: "a", Type: models.TelegramRecipient},
			true,
		},
		{
			"tg: указан чат другого типа — ошибка",
			models.Recipient{
				Name: "a",
				Type: models.TelegramRecipient,
				Chat: &models.Chat{ChType: models.ChatTypeMax},
			},
			true,
		},
		{
			"max: корректный чат — ошибки нет",
			models.Recipient{
				Name: "a",
				Type: models.MaxRecipient,
				Chat: &models.Chat{ChType: models.ChatTypeMax},
			},
			false,
		},
		{
			"email: корректный получатель — ошибки нет",
			models.Recipient{
				Name:  "a",
				Type:  models.EmailRecipient,
				Email: &models.EmailTemplate{Dest: []string{"x@y.z"}},
			},
			false,
		},
		{
			"email: шаблон не указан — ошибка",
			models.Recipient{Name: "a", Type: models.EmailRecipient},
			true,
		},
		{
			"email: получатели (dest) не указаны — ошибка",
			models.Recipient{
				Name:  "a",
				Type:  models.EmailRecipient,
				Email: &models.EmailTemplate{},
			},
			true,
		},
		{
			"smb: путь указан — ошибки нет",
			models.Recipient{Name: "a", Type: models.SambaRecipient, RemotePath: new("/x")},
			false,
		},
		{
			"smb: путь не указан — ошибка",
			models.Recipient{Name: "a", Type: models.SambaRecipient},
			true,
		},
		{
			"неподдерживаемый тип получателя — ошибка",
			models.Recipient{Name: "a", Type: "carrier_pigeon"},
			true,
		},
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

// TestValidateExport проверяет валидацию описания экспорта отчета для
// разных форматов (csv, html, text): на вход подаются корректные и неполные
// конфигурации, ожидается наличие/отсутствие ошибки валидации.
func TestValidateExport(t *testing.T) {
	v := NewReportValidation()
	ctx := context.Background()

	cases := []struct {
		name    string
		export  models.Export
		wantErr bool
	}{
		{
			"csv: имя файла указано — ошибки нет",
			models.Export{Format: models.ReportFormatCsv, FileName: new("f")},
			false,
		},
		{"csv: имя файла не указано — ошибка", models.Export{Format: models.ReportFormatCsv}, true},
		{
			"html: шаблон указан — ошибки нет",
			models.Export{
				Format:   models.ReportFormatHTML,
				FileName: new("f"),
				Template: &models.Template{TemplateText: "<p>x</p>"},
			},
			false,
		},
		{
			"html: шаблон не указан — ошибка",
			models.Export{Format: models.ReportFormatHTML, FileName: new("f")},
			true,
		},
		{
			"html: текст шаблона пустой — ошибка",
			models.Export{
				Format:   models.ReportFormatHTML,
				FileName: new("f"),
				Template: &models.Template{},
			},
			true,
		},
		{
			"text: имя файла не обязательно — ошибки нет",
			models.Export{
				Format:   models.ReportFormatText,
				Template: &models.Template{TemplateText: "x"},
			},
			false,
		},
		{"неподдерживаемый формат — ошибка", models.Export{Format: "carrier_pigeon"}, true},
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
