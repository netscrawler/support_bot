package models

import "encoding/json"

type Report struct {
	Name  string
	Title string

	Queries    []Card
	Recipients []Recipient
	Exports    []Export
	Evaluation string
}

type Card struct {
	CardUUID string `json:"card_uuid"`
	Title    string `json:"title"`
}

type RecipientType string

const (
	EmailRecipient    = "email"
	TelegramRecipient = "tg"
	SambaRecipient    = "smb"
)

type Recipient struct {
	Name       string
	Config     json.RawMessage
	RemotePath *string
	Chat       *Chat
	ThreadID   *int
	Email      *EmailTemplate
	Type       RecipientType
}

type EmailTemplate struct {
	Dest    []string
	Copy    []string
	Subject string
	Body    *string
}

type Chat struct {
	ChatID      int64
	Title       *string
	Type        string
	Description *string
	IsActive    bool
}
type Template struct {
	ID           int
	Title        string
	Type         string
	TemplateText string
}
type Export struct {
	Format   ReportFormat
	Template *Template
	FileName *string
	Order    map[string][]string
}
