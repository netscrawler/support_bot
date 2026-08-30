package layout

import (
	"testing"

	"support_bot/internal/models"
)

func tableLayout(sort []models.TableSort, limit *int) models.Layout {
	return models.Layout{
		Blocks: []models.Block{
			{
				ID:      "t",
				Type:    models.BlockTypeTable,
				Dataset: "d",
				Pos:     models.GridPosition{W: GridColumns},
				Table: &models.TableBlock{
					Columns: []models.Column{{Field: "n"}},
					Sort:    sort,
					Limit:   limit,
				},
			},
		},
	}
}

func rowNums(rows []map[string]any) []int {
	out := make([]int, len(rows))
	for i, r := range rows {
		out[i], _ = r["n"].(int)
	}

	return out
}

func TestResolveDoesNotMutateInput(t *testing.T) {
	data := map[string][]map[string]any{
		"d": {{"n": 3}, {"n": 1}, {"n": 2}},
	}
	orig := data["d"]

	doc, err := Resolve(tableLayout([]models.TableSort{{Field: "n", Desc: true}}, nil), data)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got := rowNums(doc.Blocks[0].Rows); !equalInts(got, []int{3, 2, 1}) {
		t.Fatalf("resolved rows = %v, want [3 2 1]", got)
	}

	if got := rowNums(orig); !equalInts(got, []int{3, 1, 2}) {
		t.Fatalf("input slice was reordered: %v", got)
	}

	if &doc.Blocks[0].Rows[0] == &orig[0] {
		t.Fatalf("resolved rows share the input backing array")
	}
}

func TestResolveSortIsTypedNotLexicographic(t *testing.T) {
	data := map[string][]map[string]any{
		"d": {{"n": 2}, {"n": 10}, {"n": 1}},
	}

	doc, err := Resolve(tableLayout([]models.TableSort{{Field: "n"}}, nil), data)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got := rowNums(doc.Blocks[0].Rows); !equalInts(got, []int{1, 2, 10}) {
		t.Fatalf("resolved rows = %v, want [1 2 10]", got)
	}
}

func TestResolveLimit(t *testing.T) {
	data := map[string][]map[string]any{
		"d": {{"n": 1}, {"n": 2}, {"n": 3}, {"n": 4}},
	}

	two := 2
	doc, err := Resolve(tableLayout(nil, &two), data)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got := len(doc.Blocks[0].Rows); got != 2 {
		t.Fatalf("limited rows = %d, want 2", got)
	}

	big := 99
	doc, _ = Resolve(tableLayout(nil, &big), data)
	if got := len(doc.Blocks[0].Rows); got != 4 {
		t.Fatalf("limit larger than len should be a no-op, got %d", got)
	}
}

func TestResolvePreferredAutoHeight(t *testing.T) {
	rows := make([]map[string]any, 13)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	doc, err := Resolve(tableLayout(nil, nil), map[string][]map[string]any{"d": rows})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// HeaderRows + ceil(13/6) = 1 + 3 = 4
	if got := doc.Blocks[0].PreferredSize.H; got != 4 {
		t.Fatalf("table auto height = %d, want 4", got)
	}

	if !doc.Blocks[0].Splittable {
		t.Fatalf("table block should be splittable")
	}
}

func TestResolveTextBlockAutoHeight(t *testing.T) {
	l := models.Layout{
		Blocks: []models.Block{
			{
				ID:   "x",
				Type: models.BlockTypeText,
				Pos:  models.GridPosition{W: GridColumns},
				Text: &models.TextBlock{Text: "hi"},
			},
		},
	}

	doc, err := Resolve(l, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got := doc.Blocks[0].PreferredSize.H; got != defaultAutoTextRows {
		t.Fatalf("text auto height = %d, want %d", got, defaultAutoTextRows)
	}

	if doc.Blocks[0].Splittable {
		t.Fatalf("text block must not be splittable")
	}
}

func TestResolvedBlockFromRowsClonesInput(t *testing.T) {
	rows := []map[string]any{{"n": 3}, {"n": 1}, {"n": 2}}

	block := models.Block{
		ID:   "t",
		Type: models.BlockTypeTable,
		Pos:  models.GridPosition{W: GridColumns},
		Table: &models.TableBlock{
			Columns: []models.Column{{Field: "n"}},
			Sort:    []models.TableSort{{Field: "n"}},
		},
	}

	rb := ResolvedBlockFromRows(block, rows)

	if got := rowNums(rb.Rows); !equalInts(got, []int{1, 2, 3}) {
		t.Fatalf("resolved rows = %v, want [1 2 3]", got)
	}

	if got := rowNums(rows); !equalInts(got, []int{3, 1, 2}) {
		t.Fatalf("caller slice was mutated: %v", got)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
