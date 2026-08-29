package models

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"strings"
	"text/template"
	"time"
)

// ReportStatus represents the current state of a report generation job
type ReportStatus string

const (
	ReportStatusPending    ReportStatus = "pending"
	ReportStatusProcessing ReportStatus = "processing"
	ReportStatusCompleted  ReportStatus = "completed"
	ReportStatusFailed     ReportStatus = "failed"
)

// ReportDefinition represents a generated or scheduled report with its layout configuration
// This is the new canonical model for reports (Phase 3 of EXPORT_PLAN)
type ReportDefinition struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Layout      *Layout      `json:"layout,omitempty"` // Canonical layout configuration
	Format      string       `json:"format"`           // Target format: html, pdf, xlsx, csv, png, text
	Status      ReportStatus `json:"status"`           // Current processing status
	FilePath    string       `json:"file_path,omitempty"`
	ErrorMsg    string       `json:"error_msg,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
}

// Legacy Report structure (kept for backward compatibility)
type Report struct {
	Name  string `json:"name"`
	Title string `json:"title"`

	Queries      []Card      `json:"queries"`
	Recipients   []Recipient `json:"recipients"`
	Exports      []Export    `json:"exports"`
	Pipeline     *Pipeline   `json:"pipeline,omitempty"`
	Evaluation   string      `json:"evaluation"`
	Active       bool        `json:"active"`
	AccessFromLK bool        `json:"access_from_lk"`
	Crons        []Cron      `json:"crons,omitempty"`
}

type Pipeline struct {
	Name  string `json:"name"`
	Steps []Step `json:"steps"`
}

type Step struct {
	ID         string         `json:"id,omitempty"`
	Type       string         `json:"type"`
	Script     string         `json:"script,omitempty"`
	ScriptName string         `json:"script_name,omitempty"`
	Params     map[string]any `json:"params,omitempty"`
}

type ReportInfo struct {
	ID    string
	Name  string
	Title string

	Queries    []string
	Recipients []string
	Exports    []string

	Evaluation string

	LinkedCron []string

	NextCron string
}

func (r ReportInfo) String() string {
	funcMap := template.FuncMap{
		"html": func(v any) string {
			return html.EscapeString(fmt.Sprint(v))
		},
	}

	tmpl, err := template.New("report_info").
		Funcs(funcMap).
		Parse(reportInfoStringTemplate)
	if err != nil {
		return fmt.Sprintf("Error parsing report info template: %v", err)
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, r); err != nil {
		return fmt.Sprintf("Error rendering report info template: %v", err)
	}

	return buf.String()
}

const (
	CollectTypeMetabase   = "mb"
	CollectTypeAppMetrica = "appmetrica"
	CollectTypeJIRA       = "jira"
)

type Card struct {
	CardUUID  string            `json:"card_uuid"`
	Title     string            `json:"title"`
	RawParams json.RawMessage   `json:"-"`
	Params    map[string]string `json:"params"`
	Type      string            `json:"type"`
}

func (c Card) ToString(baseUrl string) string {
	return ""
}

func (c Card) GetFullURL(baseUrl string) string {
	bUrl := strings.TrimRight(baseUrl, "/")

	return fmt.Sprintf("%s/public/question/%s", bUrl, c.CardUUID)
}

type Evaluator interface {
	EvalStr(ctx context.Context, expr string) (string, error)
}

func (c *Card) ResolveParams(ctx context.Context, eval Evaluator) error {
	var params map[string]string

	c.Params = make(map[string]string)

	err := json.Unmarshal(c.RawParams, &params)
	if err != nil {
		return err
	}

	var evErr error

	for k, v := range params {
		q, err := eval.EvalStr(ctx, v)
		if err != nil {
			evErr = errors.Join(evErr, err)
		}

		c.Params[k] = q
	}

	return evErr
}

type RecipientType string

const (
	emailRecipient    = "email"
	TelegramRecipient = "tg"
	sambaRecipient    = "smb"
	maxRecipient      = "max"
)

type Recipient struct {
	Name       string          `json:"name"`
	Config     json.RawMessage `json:"config"`
	RemotePath *string         `json:"remote_path,omitempty"`
	Chat       *Chat           `json:"chat,omitempty"`
	ThreadID   *int            `json:"thread_id,omitempty"`
	Email      *EmailTemplate  `json:"email,omitempty"`
	Type       RecipientType   `json:"type"`

	NeedDeleteAfterEndOfDay bool `json:"need_delete_after_end_of_day"`
}

func (r Recipient) String() string {
	funcMap := template.FuncMap{
		"html": func(v any) string {
			return html.EscapeString(fmt.Sprint(v))
		},
	}

	tmpl, err := template.New("recipient").
		Funcs(funcMap).
		Parse(recipientStringTemplate)
	if err != nil {
		return fmt.Sprintf("Error parsing recipient template: %v", err)
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, r); err != nil {
		return fmt.Sprintf("Error rendering recipient template: %v", err)
	}

	return buf.String()
}

type EmailTemplate struct {
	Dest    []string `json:"dest"`
	Copy    []string `json:"copy"`
	Subject string   `json:"subject"`
	Body    *string  `json:"body,omitempty"`
}

const (
	ChatTypeTg  = "tg"
	ChatTypeMax = "max"
)

type Chat struct {
	ChatID      int64   `json:"chat_id"`
	Title       *string `json:"title,omitempty"`
	Type        string  `json:"type"`
	Description *string `json:"description,omitempty"`
	IsActive    bool    `json:"is_active"`
	ChType      string  `json:"ch_type"`
}
type Template struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	TemplateText string `json:"template_text"`
}

type Cron struct {
	Name        string `json:"name"`
	Cron        string `json:"cron"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	EventType   int    `json:"event_type"`
}
type Export struct {
	Format   reportFormat        `json:"format"`
	Template *Template           `json:"template,omitempty"`
	FileName *string             `json:"file_name,omitempty"`
	Order    map[string][]string `json:"order,omitempty"`
	Layout   *Layout             `json:"layout,omitempty"`
}

func (e Export) String() string {
	tmpl, err := template.New("export").
		Parse(exportStringTemplate)
	if err != nil {
		return fmt.Sprintf("Error parsing export template: %v", err)
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, e); err != nil {
		return fmt.Sprintf("Error rendering export template: %v", err)
	}

	return buf.String()
}
