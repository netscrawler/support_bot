package models_test

import (
	"encoding/json"
	"strings"
	"testing"

	"support_bot/internal/models"
)

func TestReportJSON(t *testing.T) {
	rep := models.Report{
		Name:  "TEST",
		Title: "TestReport",
		Queries: []models.Card{
			{
				CardUUID: "1f2ff9eb-6f58-45e0-9d3f-3b6e3780d670",
				Title:    "test_query",
				Params: map[string]string{
					"data":  "20.07.2026",
					"data2": "21.07.2026",
				},
				Type: "mb",
			},
			{
				CardUUID: "a45ad11b-4be8-4531-86a0-3ecf4c386efe",
				Title:    "test_query2",
				Params:   map[string]string{},
				Type:     "jira",
			},
		},
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

func TestReportJSON_IgnoreEvaluateNotSerialized(t *testing.T) {
	b, err := json.Marshal(models.Report{Name: "r", IgnoreEvaluate: true})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(strings.ToLower(string(b)), "ignore") {
		t.Fatalf("IgnoreEvaluate leaked into JSON: %s", b)
	}
}

func TestReportJSON_Dedup(t *testing.T) {
	var rep models.Report

	if err := json.Unmarshal(
		[]byte(`{"name":"a"}`),
		&rep,
	); err != nil ||
		rep.Dedup {
		t.Fatalf(
			"without dedup: Dedup = %v, err = %v, want false",
			rep.Dedup,
			err,
		)
	}

	if err := json.Unmarshal(
		[]byte(`{"name":"a","dedup":true}`),
		&rep,
	); err != nil ||
		!rep.Dedup {
		t.Fatalf("dedup:true: Dedup = %v, err = %v, want true", rep.Dedup, err)
	}

	b, err := json.Marshal(rep)
	if err != nil || !strings.Contains(string(b), `"dedup":true`) {
		t.Errorf("Marshal = %s, err = %v, want contains \"dedup\":true", b, err)
	}
}

func TestReport_UsesPeriod(t *testing.T) {
	with := models.Report{
		Queries: []models.Card{{RawParams: []byte(`{"d":"period_from"}`)}},
	}
	without := models.Report{
		Queries: []models.Card{{RawParams: []byte(`{"d":"today"}`)}},
	}

	if !with.UsesPeriod() || without.UsesPeriod() {
		t.Errorf(
			"UsesPeriod: with=%v without=%v",
			with.UsesPeriod(),
			without.UsesPeriod(),
		)
	}
}
