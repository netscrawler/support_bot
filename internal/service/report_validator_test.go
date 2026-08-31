package service

import (
	"testing"

	"support_bot/internal/models"
)

func reportWithTableRules(rules []models.CondRule) models.Report {
	return models.Report{
		Name:       "r",
		Title:      "t",
		Evaluation: "[*]",
		Queries:    []models.Card{{Title: "daily", Type: "mb", CardUUID: "u"}},
		Exports: []models.Export{{
			Format: "html",
			Layout: &models.Layout{
				Version: 1,
				Blocks: []models.Block{{
					ID:      "t",
					Type:    models.BlockTypeTable,
					Dataset: "daily",
					Pos:     models.GridPosition{W: 12},
					Table: &models.TableBlock{
						Columns:     []models.Column{{Field: "accepted"}},
						Conditional: rules,
					},
				}},
			},
		}},
	}
}

func TestValidateLayoutCompilesConditionalRules(t *testing.T) {
	v := NewReportValidation()

	ok := reportWithTableRules([]models.CondRule{{
		When:    "has(row.accepted) && value < 100",
		Target:  models.CondTargetCell,
		Columns: []string{"accepted"},
		Style:   models.CondStyleBad,
	}})
	if err := v.Validate(t.Context(), ok); err != nil {
		t.Fatalf("valid rule rejected: %v", err)
	}

	bad := reportWithTableRules([]models.CondRule{{
		When:   "value <",
		Target: models.CondTargetRow,
		Style:  models.CondStyleBad,
	}})
	if err := v.Validate(t.Context(), bad); err == nil {
		t.Fatal("malformed rule expression passed validation")
	}
}
