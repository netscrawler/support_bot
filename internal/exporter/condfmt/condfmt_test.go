package condfmt_test

import (
	"context"
	"testing"

	"support_bot/internal/evaluator"
	"support_bot/internal/exporter/condfmt"
	"support_bot/internal/models"
)

func newEngine(t *testing.T) *evaluator.RowEngine {
	t.Helper()

	e, err := evaluator.NewRowEngine()
	if err != nil {
		t.Fatalf("new row engine: %v", err)
	}

	return e
}

func TestNewNoRules(t *testing.T) {
	f, err := condfmt.New(nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f != nil {
		t.Fatal("expected nil formatter for an empty rule set")
	}

	// A nil formatter is inert.
	if got := f.Row(context.Background(), map[string]any{}); got.Set {
		t.Errorf("nil formatter returned a set decoration: %+v", got)
	}
}

func TestCellRuleOutranksRowRule(t *testing.T) {
	rows := []map[string]any{
		{"channel": "email", "share": 0.05},
		{"channel": "web", "share": 0.60},
	}

	rules := []models.CondRule{
		{When: "true", Target: models.CondTargetRow, Style: models.CondStyleMuted},
		{
			When:    "value < 0.1",
			Target:  models.CondTargetCell,
			Columns: []string{"share"},
			Style:   models.CondStyleBad,
		},
	}

	f, err := condfmt.New(newEngine(t), rules, rows)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()

	rowDec := f.Row(ctx, rows[0])
	if rowDec.BG != "#F1F3F5" {
		t.Fatalf("row rule not applied: %+v", rowDec)
	}

	// share 0.05 -> the cell rule wins over the row rule.
	share := f.Cell(ctx, "share", rows[0], rowDec)
	if share.BG != "#FDECEA" {
		t.Errorf("cell rule did not out-rank the row rule: %+v", share)
	}

	// channel has no cell rule -> it keeps the row decoration.
	channel := f.Cell(ctx, "channel", rows[0], rowDec)
	if channel.BG != "#F1F3F5" {
		t.Errorf("cell without its own rule lost the row decoration: %+v", channel)
	}

	// share 0.60 -> the cell rule does not match, row decoration remains.
	share2 := f.Cell(ctx, "share", rows[1], f.Row(ctx, rows[1]))
	if share2.BG != "#F1F3F5" {
		t.Errorf("non-matching cell rule changed the decoration: %+v", share2)
	}
}

func TestExplicitColoursOverridePreset(t *testing.T) {
	rows := []map[string]any{{"n": 1}}

	rules := []models.CondRule{{
		When:   "true",
		Target: models.CondTargetRow,
		Style:  models.CondStyleGood,
		BG:     "#123456",
		Bold:   true,
	}}

	f, err := condfmt.New(newEngine(t), rules, rows)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	got := f.Row(context.Background(), rows[0])
	if got.BG != "#123456" {
		t.Errorf("explicit bg did not override the preset: %+v", got)
	}

	if got.FG != "#1E7E34" {
		t.Errorf("preset fg should remain when only bg is overridden: %+v", got)
	}

	if !got.Bold {
		t.Errorf("explicit bold lost: %+v", got)
	}
}

func TestLastMatchWins(t *testing.T) {
	rows := []map[string]any{{"n": 5}}

	cell := models.CondTargetCell
	cols := []string{"n"}
	rules := []models.CondRule{
		{When: "value > 0", Target: cell, Columns: cols, Style: models.CondStyleGood},
		{When: "value > 3", Target: cell, Columns: cols, Style: models.CondStyleBad},
	}

	f, err := condfmt.New(newEngine(t), rules, rows)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	got := f.Cell(context.Background(), "n", rows[0], condfmt.Decoration{})
	if got.BG != "#FDECEA" {
		t.Errorf("last matching rule should win: %+v", got)
	}
}

func TestBrokenRuleWarnsAndDoesNotMatch(t *testing.T) {
	rows := []map[string]any{{"n": 1}, {"n": 2}}

	rules := []models.CondRule{{
		When:    "value > 0 && row.missing > 0",
		Target:  models.CondTargetCell,
		Columns: []string{"n"},
		Style:   models.CondStyleBad,
	}}

	f, err := condfmt.New(newEngine(t), rules, rows)
	if err != nil {
		t.Fatalf("new: %v", err)
	}

	ctx := context.Background()
	for _, r := range rows {
		if got := f.Cell(ctx, "n", r, condfmt.Decoration{}); got.Set {
			t.Errorf("broken rule produced a decoration: %+v", got)
		}
	}

	if w := f.Warnings(); len(w) != 1 {
		t.Fatalf("want one warning logged once, got %d: %v", len(w), w)
	}
}

func TestCompileErrorIsFatal(t *testing.T) {
	rules := []models.CondRule{{
		When:   "value <",
		Target: models.CondTargetRow,
		Style:  models.CondStyleBad,
	}}

	if _, err := condfmt.New(newEngine(t), rules, nil); err == nil {
		t.Fatal("expected a compile error for a malformed expression")
	}
}
