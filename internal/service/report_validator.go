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
	}

	for _, exp := range report.Exports {
		if exp.Format == "" {
			return fmt.Errorf("export format is empty in report %s", report.Name)
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

func (r *ReportValidation) validateReport(ctx context.Context, report models.Report) error {
	return nil
}

func (r *ReportValidation) validateQuery(ctx context.Context, query models.Card) error {
	return nil
}

func (r *ReportValidation) validateQueryParams(
	ctx context.Context,
	params map[string]string,
) error {
	return nil
}

func (r *ReportValidation) validateRecipient(
	ctx context.Context,
	recipient models.Recipient,
) error {
	return nil
}

func (r *ReportValidation) validateRecipientRemotePath(
	ctx context.Context,
	remotePath string,
) error {
	return nil
}

func (r *ReportValidation) validateRecipientChat(ctx context.Context, ch models.Chat) error {
	return nil
}

func (r *ReportValidation) validateRecipientEmailTemplate(
	ctx context.Context,
	tmpl models.EmailTemplate,
) error {
	return nil
}

func (r *ReportValidation) validateExport(ctx context.Context, export models.Export) error {
	return nil
}

func (r *ReportValidation) validateExportTemplate(ctx context.Context, tmpl models.Template) error {
	return nil
}

func (r *ReportValidation) validatePipeline(
	ctx context.Context,
	pipeline models.Pipeline,
) error {
	return nil
}

func (r *ReportValidation) validateEvaluation(
	ctx context.Context,
	evaluation string,
) error {
	return nil
}
