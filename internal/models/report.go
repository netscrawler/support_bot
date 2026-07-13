package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"text/template"
)

type Report struct {
	Name  string
	Title string

	Queries    []Card
	Recipients []Recipient
	Exports    []Export
	Evaluation string

	Workflow *json.RawMessage
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

type Card struct {
	CardUUID string `json:"card_uuid"`
	Title    string `json:"title"`
}

func (c Card) GetFullURL(baseUrl string) string {
	bUrl := strings.TrimRight(baseUrl, "/")
	return fmt.Sprintf("%s/public/question/%s", bUrl, c.CardUUID)
}

type RecipientType string

const (
	emailRecipient    = "email"
	TelegramRecipient = "tg"
	sambaRecipient    = "smb"
	maxRecipient      = "max"
)

type Recipient struct {
	Name       string
	Config     json.RawMessage
	RemotePath *string
	Chat       *Chat
	ThreadID   *int
	Email      *EmailTemplate
	Type       RecipientType

	NeedDeleteAfterEndOfDay bool
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
	Dest    []string
	Copy    []string
	Subject string
	Body    *string
}

const (
	ChatTypeTg  = "tg"
	ChatTypeMax = "max"
)

type Chat struct {
	ChatID      int64
	Title       *string
	Type        string
	Description *string
	IsActive    bool
	ChType      string
}
type Template struct {
	ID           int
	Title        string
	Type         string
	TemplateText string
}
type Export struct {
	Format   reportFormat
	Template *Template
	FileName *string
	Order    map[string][]string
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
