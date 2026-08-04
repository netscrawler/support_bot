package models_test

import (
	"encoding/json"
	"testing"

	"support_bot/internal/models"
)

func TestReportJSON(t *testing.T) {
	rep := models.Report{
		Name:  "TEST",
		Title: "TestReport",
		Queries: []models.Card{{
			CardUUID: "1f2ff9eb-6f58-45e0-9d3f-3b6e3780d670",
			Title:    "test_query",
			Params:   map[string]string{"data": "20.07.2026", "data2": "21.07.2026"},
			Type:     "mb",
		}, {
			CardUUID: "a45ad11b-4be8-4531-86a0-3ecf4c386efe",
			Title:    "test_query2",
			Params:   map[string]string{},
			Type:     "jira",
		}},
		Recipients: []models.Recipient{{
			Name:       "RCPT1",
			RemotePath: new(string),
			Chat: &models.Chat{
				ChatID:      0,
				Title:       new(string),
				Type:        "group",
				Description: new(string),
				IsActive:    false,
				ChType:      "tg",
			},
			ThreadID: new(int),
			Email: &models.EmailTemplate{
				Dest:    []string{"hello"},
				Copy:    []string{"hello"},
				Subject: "Subj",
				Body:    new(string),
			},
			Type:                    "",
			NeedDeleteAfterEndOfDay: false,
		}},
		Exports: []models.Export{{
			Format: "",
			Template: &models.Template{
				ID:           0,
				Title:        "test",
				Type:         "rich_text",
				TemplateText: "jfsjkjsfkj",
			},
			FileName: new(string),
			Order:    map[string][]string{},
		}, {
			Format:   "",
			Template: &models.Template{},
			FileName: new(string),
			Order:    map[string][]string{},
		}},
		Pipeline: &models.Pipeline{
			Name: "",
			Steps: []models.Step{{
				ID:         "",
				Type:       "",
				Script:     "",
				ScriptName: "",
				Params:     map[string]any{},
			}, {
				ID:         "",
				Type:       "",
				Script:     "",
				ScriptName: "",
				Params:     map[string]any{},
			}},
		},
		Evaluation: "",
	}

	jR, _ := json.Marshal(rep)
	t.Log(string(jR))
}
