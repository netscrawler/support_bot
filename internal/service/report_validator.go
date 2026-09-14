package service

import (
	"context"
	"fmt"
	"support_bot/internal/models"
)

type ReportValidation struct{}

func NewReportValidation() *ReportValidation {
	return &ReportValidation{}
}

func (r *ReportValidation) Validate(ctx context.Context, report models.Report) error {
	if report.Name == "" {
		return fmt.Errorf("report name is empty")
	}

	if report.Title == "" {
		return fmt.Errorf("report title is empty")
	}

	if report.Evaluation == "" {
		return fmt.Errorf("report evaluation is empty")
	}

	seenQueries := make(map[string]bool)

	for _, q := range report.Queries {
		if q.Title == "" {
			return fmt.Errorf("query title is empty in report %s", report.Name)
		}

		if seenQueries[q.Title] {
			return fmt.Errorf("duplicate query title %q in report %s", q.Title, report.Name)
		}

		seenQueries[q.Title] = true
		if q.CardUUID == "" && q.Type == "mb" {
			return fmt.Errorf("query %q card_uuid is empty", q.Title)
		}
	}

	seenRecipients := make(map[string]bool)

	for _, rec := range report.Recipients {
		if rec.Name == "" {
			return fmt.Errorf("recipient name is empty in report %s", report.Name)
		}

		if seenRecipients[rec.Name] {
			return fmt.Errorf("duplicate recipient name %q in report %s", rec.Name, report.Name)
		}

		seenRecipients[rec.Name] = true
		if rec.Type == "" {
			return fmt.Errorf("recipient %q type is empty", rec.Name)
		}

		if err := r.validateRecipient(ctx, rec); err != nil {
			return fmt.Errorf("recipient %q: %w", rec.Name, err)
		}
	}

	for _, exp := range report.Exports {
		if exp.Format == "" {
			return fmt.Errorf("export format is empty in report %s", report.Name)
		}

		if err := r.validateExport(ctx, exp); err != nil {
			return fmt.Errorf("export %q in report %s: %w", exp.Format, report.Name, err)
		}
	}

	seenCrons := make(map[string]bool)

	for _, c := range report.Crons {
		if c.Name == "" {
			return fmt.Errorf("cron name is empty in report %s", report.Name)
		}

		if c.Cron == "" {
			return fmt.Errorf("cron expression is empty for %q in report %s", c.Name, report.Name)
		}

		key := fmt.Sprintf("%s:%s", c.Name, c.Cron)
		if seenCrons[key] {
			return fmt.Errorf(
				"duplicate cron %q with expr %q in report %s",
				c.Name,
				c.Cron,
				report.Name,
			)
		}

		seenCrons[key] = true
	}

	if report.Pipeline != nil {
		if len(report.Pipeline.Steps) == 0 {
			return fmt.Errorf("pipeline in report %s has no steps", report.Name)
		}
	}

	return nil
}

// validateRecipient checks that a recipient carries the fields its delivery
// channel (models.Message.Send) will dereference unconditionally, so a bad
// report fails validation instead of panicking at send time.
func (r *ReportValidation) validateRecipient(
	ctx context.Context,
	recipient models.Recipient,
) error {
	switch recipient.Type {
	case models.TelegramRecipient:
		return r.validateRecipientChat(ctx, recipient.Chat, models.ChatTypeTg)
	case models.MaxRecipient:
		return r.validateRecipientChat(ctx, recipient.Chat, models.ChatTypeMax)
	case models.EmailRecipient:
		return r.validateRecipientEmailTemplate(ctx, recipient.Email)
	case models.SambaRecipient:
		return r.validateRecipientRemotePath(ctx, recipient.RemotePath)
	default:
		return fmt.Errorf("unsupported recipient type %q", recipient.Type)
	}
}

func (r *ReportValidation) validateRecipientRemotePath(
	ctx context.Context,
	remotePath *string,
) error {
	if remotePath == nil || *remotePath == "" {
		return fmt.Errorf("remote_path is required for smb recipients")
	}

	return nil
}

func (r *ReportValidation) validateRecipientChat(
	ctx context.Context,
	ch *models.Chat,
	wantChatType string,
) error {
	if ch == nil {
		return fmt.Errorf("chat is required for %s recipients", wantChatType)
	}

	if ch.ChType != wantChatType {
		return fmt.Errorf("chat type %q does not match recipient type %q", ch.ChType, wantChatType)
	}

	return nil
}

func (r *ReportValidation) validateRecipientEmailTemplate(
	ctx context.Context,
	tmpl *models.EmailTemplate,
) error {
	if tmpl == nil {
		return fmt.Errorf("email is required for email recipients")
	}

	if len(tmpl.Dest) == 0 {
		return fmt.Errorf("email recipient has no destination addresses")
	}

	return nil
}

// validateExport checks fields that internal/exporter.Export dereferences
// unconditionally per format, so a bad report fails validation instead of
// panicking at export time.
func (r *ReportValidation) validateExport(ctx context.Context, export models.Export) error {
	switch export.Format {
	case models.ReportFormatCsv, models.ReportFormatXlsx, models.ReportFormatPng,
		models.ReportFormatHTML, models.ReportFormatPdf:
		if export.FileName == nil || *export.FileName == "" {
			return fmt.Errorf("file_name is required for format %q", export.Format)
		}
	case models.ReportFormatText:
	default:
		return fmt.Errorf("unsupported export format %q", export.Format)
	}

	switch export.Format {
	case models.ReportFormatHTML, models.ReportFormatPdf, models.ReportFormatText:
		return r.validateExportTemplate(ctx, export.Template)
	default:
		return nil
	}
}

func (r *ReportValidation) validateExportTemplate(
	ctx context.Context,
	tmpl *models.Template,
) error {
	if tmpl == nil {
		return fmt.Errorf("template is required for this export format")
	}

	if tmpl.TemplateText == "" {
		return fmt.Errorf("template %q has empty template_text", tmpl.Title)
	}

	return nil
}
