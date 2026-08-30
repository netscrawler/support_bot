package layout

import (
	"testing"

	"support_bot/internal/models"
)

func chartBlock(id string, w, h int) models.Block {
	return models.Block{
		ID:      id,
		Type:    models.BlockTypeChart,
		Dataset: "d",
		Pos:     models.GridPosition{W: w, H: h},
		Chart: &models.ChartBlock{
			Kind:   models.ChartKindLine,
			XField: "x",
			Series: []models.Series{{Field: "y"}},
		},
	}
}

func resolvedDoc(t *testing.T, blocks ...models.Block) ResolvedDocument {
	t.Helper()

	data := map[string][]map[string]any{"d": {{"x": 1, "y": 1}}}

	doc, err := Resolve(models.Layout{Blocks: blocks}, data)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	return doc
}

func TestPackUnboundedSinglePage(t *testing.T) {
	doc := resolvedDoc(t, chartBlock("a", 4, 4), chartBlock("b", 4, 4))

	pages, err := Pack(doc, PageGeometry{Cols: GridColumns, Rows: 0})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if len(pages) != 1 || len(pages[0].Blocks) != 2 {
		t.Fatalf("want 1 page / 2 blocks, got %d pages", len(pages))
	}

	if pages[0].Blocks[0].Pos != doc.Blocks[0].PreferredSize {
		t.Fatalf("unbounded pack must keep PreferredSize, got %+v", pages[0].Blocks[0].Pos)
	}
}

func TestPackRespectAuthoredPos(t *testing.T) {
	b := chartBlock("a", 4, 4)
	b.Pos = models.GridPosition{X: 2, Y: 3, W: 5, H: 6}

	pages, err := Pack(resolvedDoc(t, b), PageGeometry{RespectAuthoredPos: true})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if got := pages[0].Blocks[0].Pos; got != (models.GridPosition{X: 2, Y: 3, W: 5, H: 6}) {
		t.Fatalf("authored pos not preserved: %+v", got)
	}
}

func TestPackFirstFitRowMajor(t *testing.T) {
	doc := resolvedDoc(t, chartBlock("a", 4, 4), chartBlock("b", 4, 4), chartBlock("c", 4, 4))

	pages, err := Pack(doc, PageGeometry{Cols: GridColumns, Rows: GridRows})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("3 blocks of w=4 fit one row, got %d pages", len(pages))
	}

	wantX := []int{0, 4, 8}
	for i, pb := range pages[0].Blocks {
		if pb.Pos.X != wantX[i] || pb.Pos.Y != 0 {
			t.Fatalf("block %d at (%d,%d), want (%d,0)", i, pb.Pos.X, pb.Pos.Y, wantX[i])
		}
	}
}

func TestPackOverflowToNextPage(t *testing.T) {
	// two full-height blocks cannot share a page
	doc := resolvedDoc(t, chartBlock("a", 12, 12), chartBlock("b", 12, 12))

	pages, err := Pack(doc, PageGeometry{Cols: GridColumns, Rows: GridRows})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if len(pages) != 2 {
		t.Fatalf("want 2 pages, got %d", len(pages))
	}

	if pages[1].Blocks[0].PageIndex != 1 {
		t.Fatalf("second block PageIndex = %d, want 1", pages[1].Blocks[0].PageIndex)
	}
}

func TestPackTableSplitsAcrossPages(t *testing.T) {
	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	doc, err := Resolve(tableLayout(nil, nil), map[string][]map[string]any{"d": rows})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	pages, err := Pack(doc, PageGeometry{Cols: GridColumns, Rows: GridRows})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if len(pages) < 2 {
		t.Fatalf("200-row table should span >= 2 pages, got %d", len(pages))
	}

	total, nextOffset := 0, 0
	for _, p := range pages {
		for _, pb := range p.Blocks {
			if pb.RowOffset != nextOffset {
				t.Fatalf("non-contiguous chunk: offset %d, want %d", pb.RowOffset, nextOffset)
			}

			if pb.Pos.H > GridRows {
				t.Fatalf("chunk height %d exceeds page", pb.Pos.H)
			}

			nextOffset += pb.RowCount
			total += pb.RowCount
		}
	}

	if total != 200 {
		t.Fatalf("chunks cover %d rows, want 200", total)
	}
}

func TestPackWidthOutOfRange(t *testing.T) {
	b := chartBlock("a", 4, 4)
	b.Pos.W = 13

	_, err := Pack(resolvedDoc(t, b), PageGeometry{Cols: GridColumns, Rows: GridRows})
	if err == nil {
		t.Fatalf("expected width error")
	}
}

func TestPackNonSplittableTooTall(t *testing.T) {
	b := chartBlock("a", 4, 13)

	_, err := Pack(resolvedDoc(t, b), PageGeometry{Cols: GridColumns, Rows: GridRows})
	if err == nil {
		t.Fatalf("expected height error for non-splittable block")
	}
}
