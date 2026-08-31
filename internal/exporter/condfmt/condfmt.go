// Package condfmt applies a table block's conditional-format rules, turning
// each into a format-neutral Decoration for a row or a cell. Rule expressions
// are CEL, evaluated through evaluator.RowEngine, so the html and xlsx exporters
// share one implementation and one set of colours.
package condfmt

import (
	"context"
	"fmt"
	"strings"

	"support_bot/internal/evaluator"
	"support_bot/internal/models"
)

// Decoration is the styling a matched rule applies. Set is false when no rule
// matched (the zero value).
type Decoration struct {
	BG     string
	FG     string
	Bold   bool
	Italic bool
	Set    bool
}

type rule struct {
	when    string
	isRow   bool
	columns map[string]struct{} // nil for a row rule
	dec     Decoration
}

// Formatter holds one table block's compiled rules. A nil *Formatter is valid
// and inert: New returns nil when the block declares no rules, and every method
// tolerates that.
type Formatter struct {
	engine *evaluator.RowEngine
	rules  []rule
	rows   []map[string]any

	seen     map[int]struct{}
	warnings []string
}

// New compiles the rules of one table block against the shared engine. rows is
// the block's full row set — never a paginated window — so a `rows` aggregate
// evaluates the same in every format. It returns (nil, nil) when rules is
// empty.
func New(
	engine *evaluator.RowEngine,
	rules []models.CondRule,
	rows []map[string]any,
) (*Formatter, error) {
	if len(rules) == 0 {
		return nil, nil
	}

	if engine == nil {
		return nil, fmt.Errorf("condfmt: %d rule(s) but no engine", len(rules))
	}

	f := &Formatter{engine: engine, rows: rows, seen: make(map[int]struct{})}

	for _, r := range rules {
		if err := engine.Compile(r.When); err != nil {
			return nil, err
		}

		cr := rule{
			when:  r.When,
			isRow: r.Target == models.CondTargetRow,
			dec:   decorationFor(r),
		}

		if !cr.isRow {
			cr.columns = make(map[string]struct{}, len(r.Columns))
			for _, c := range r.Columns {
				cr.columns[strings.TrimSpace(c)] = struct{}{}
			}
		}

		f.rules = append(f.rules, cr)
	}

	return f, nil
}

// Row returns the row-level decoration: the last matching row-target rule, or
// the zero Decoration.
func (f *Formatter) Row(ctx context.Context, row map[string]any) Decoration {
	if f == nil {
		return Decoration{}
	}

	var out Decoration

	for i := range f.rules {
		if f.rules[i].isRow && f.match(ctx, i, row, nil) {
			out = f.rules[i].dec
		}
	}

	return out
}

// Cell returns the decoration for one field: the last matching cell rule for
// that column, else rowDec (obtained from Row). A cell rule out-ranks a row
// rule on the same cell.
func (f *Formatter) Cell(
	ctx context.Context,
	field string,
	row map[string]any,
	rowDec Decoration,
) Decoration {
	if f == nil {
		return Decoration{}
	}

	out := rowDec

	for i := range f.rules {
		r := &f.rules[i]
		if r.isRow {
			continue
		}

		if _, ok := r.columns[field]; !ok {
			continue
		}

		if f.match(ctx, i, row, row[field]) {
			out = r.dec
		}
	}

	return out
}

// Warnings returns one message per rule that failed to evaluate at least once
// (a missing key, a type clash). Such a rule simply does not match; delivery is
// never blocked. The caller may log these.
func (f *Formatter) Warnings() []string {
	if f == nil {
		return nil
	}

	return f.warnings
}

func (f *Formatter) match(ctx context.Context, idx int, row map[string]any, value any) bool {
	ok, err := f.engine.Match(ctx, f.rules[idx].when, row, value, f.rows)
	if err != nil {
		if _, done := f.seen[idx]; !done {
			f.seen[idx] = struct{}{}
			f.warnings = append(f.warnings, err.Error())
		}

		return false
	}

	return ok
}

// presetColours maps a semantic style to its background / foreground.
var presetColours = map[models.CondStyle]Decoration{
	models.CondStyleGood:    {BG: "#E6F4EA", FG: "#1E7E34", Set: true},
	models.CondStyleWarning: {BG: "#FFF4E5", FG: "#A15C00", Set: true},
	models.CondStyleBad:     {BG: "#FDECEA", FG: "#B42318", Set: true},
	models.CondStyleMuted:   {BG: "#F1F3F5", FG: "#6B7280", Set: true},
}

// decorationFor resolves a rule to its concrete Decoration: the preset, then
// any explicit bg / fg / bold / italic layered on top.
func decorationFor(r models.CondRule) Decoration {
	d := presetColours[r.Style]
	d.Set = true

	if r.BG != "" {
		d.BG = r.BG
	}

	if r.FG != "" {
		d.FG = r.FG
	}

	if r.Bold {
		d.Bold = true
	}

	if r.Italic {
		d.Italic = true
	}

	return d
}
