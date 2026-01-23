package orchestrator

import (
	"encoding/json"
	"errors"
	"strings"

	models "support_bot/internal/models/report"
)

type report struct {
	ID    int    `db:"id"`
	Name  string `db:"name"`
	Title string `db:"title"`
	Expr  string `db:"evaluation"`
}

type card struct {
	CardUUID string `db:"card_uuid"`
	Title    string `db:"title"`
}

func mapCardToModel(c card) models.Card {
	return models.Card{
		CardUUID: c.CardUUID,
		Title:    c.Title,
	}
}

func mapCardsToModels(c ...card) []models.Card {
	var crds []models.Card

	for _, crd := range c {
		crds = append(crds, mapCardToModel(crd))
	}

	return crds
}

type recipient struct {
	Name string `db:"name"`

	Config json.RawMessage `db:"config"`

	RemotePath *string `db:"remote_path"`

	EmailID *int    `db:"email_id"`
	Dest    *string `db:"dest"`
	Copy    *string `db:"copy"`
	Subject *string `db:"subject"`
	Body    *string `db:"body"`
	Type    string  `db:"type"`

	ChatID      *int64  `db:"chat_id"`
	ThreadID    *int    `db:"thread_id"`
	ChatTitle   *string `db:"title"`
	ChatType    *string `db:"chat_type"`
	Description *string `db:"description"`
	IsActive    *bool   `db:"is_active"`
}

func deref[T any](t *T) T {
	if t != nil {
		return *t
	}

	return *new(T)
}

func mapRecipientToModel(r recipient) models.Recipient {
	var c *models.Chat

	if r.ChatID != nil {
		c = &models.Chat{
			ChatID:      *r.ChatID,
			Title:       r.ChatTitle,
			Type:        deref(r.ChatType),
			Description: r.Description,
			IsActive:    *r.IsActive,
		}
	}

	var e *models.EmailTemplate

	var dest []string

	if r.Copy != nil {
		dest = pqArrayToArray(*r.Dest)
	}

	var rCopy []string

	if r.Copy != nil {
		rCopy = pqArrayToArray(*r.Copy)
	}

	if r.EmailID != nil {
		e = &models.EmailTemplate{
			Dest:    dest,
			Copy:    rCopy,
			Subject: deref(r.Subject),
			Body:    r.Body,
		}
	}

	return models.Recipient{
		Name:       r.Name,
		Config:     r.Config,
		RemotePath: r.RemotePath,
		Chat:       c,
		ThreadID:   r.ThreadID,
		Email:      e,
		Type:       models.RecipientType(r.Type),
	}
}

func pqArrayToArray(arr string) []string {
	a := strings.TrimLeft(arr, "{")
	a = strings.TrimRight(a, "}")

	return strings.Split(a, ",")
}

func mapRecipientsToModel(r ...recipient) []models.Recipient {
	var rcpts []models.Recipient

	for _, rcpt := range r {
		rcpts = append(rcpts, mapRecipientToModel(rcpt))
	}

	return rcpts
}

type export struct {
	Format   string  `db:"format"`
	FileName *string `db:"file_name"`

	TemplateID    *int            `db:"id"`
	TemplateTitle *string         `db:"title"`
	TemplateType  *string         `db:"type"`
	TemplateText  *string         `db:"template_text"`
	Order         json.RawMessage `db:"sort_order"`
}

func mapExportToModel(e export) (models.Export, error) {
	var t *models.Template

	if e.TemplateID != nil {
		t = &models.Template{
			ID:           *e.TemplateID,
			Title:        *e.TemplateTitle,
			Type:         *e.TemplateType,
			TemplateText: *e.TemplateText,
		}
	}

	var order map[string][]string

	exp := models.Export{
		Format:   e.Format,
		Template: t,
		FileName: e.FileName,
		Order:    map[string][]string{},
	}
	if e.Order != nil {
		err := json.Unmarshal(e.Order, &order)
		if err != nil {
			return exp, err
		}
	}

	exp.Order = order

	return exp, nil
}

func mapExportsToModel(e ...export) ([]models.Export, error) {
	var (
		mapErr error
		exprts []models.Export
	)

	for _, exp := range e {
		expr, err := mapExportToModel(exp)
		if err != nil {
			mapErr = errors.Join(mapErr, err)
		}

		exprts = append(exprts, expr)
	}

	return exprts, mapErr
}
