package tests

import (
	"testing"

	"support_bot/internal/models"
)

func TestFullWorkflow(t *testing.T) {
	t.Parallel()
	t.Run("simple", func(t *testing.T) {
		workfl := []byte(`{
		"id":"test-1",
		"version":"v1",
		"nodes":[
		{
		"id":"n1",
		"type":"std@noop",
		"config":{"limit":10}
		}
		],
		"edges":[],
		"metadata":{"env":"test"}
	}`)

		report := models.Report{
			Name:       "TEST",
			Title:      "TEST_REPORT",
			Queries:    []models.Card{},
			Recipients: nil,
			Exports: []models.Export{{
				Format:   models.ReportFormatCsv,
				FileName: ptr("test.csv"),
			}},
			Evaluation: "[*]",
			Workflow:   nil,
		}
	})
}

func deref[T any](t *T) T {
	var tt T
	if t == nil {
		return tt
	}
	return *t
}

func ptr[T any](t T) *T {
	return &t
}
