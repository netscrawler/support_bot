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

func flowGeometry() PageGeometry {
	g := PageGeometryFor(models.PageConfig{})
	g.Flow = true

	return g
}

func TestPackFlowWrapsAndFillsWidth(t *testing.T) {
	// 8 + 6 overflow one row: the 8 stretches to 12, the 6 wraps and also
	// stretches to 12.
	doc := resolvedDoc(t, chartBlock("a", 8, 4), chartBlock("b", 6, 4))

	pages, err := Pack(doc, flowGeometry())
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if len(pages) != 1 || len(pages[0].Blocks) != 2 {
		t.Fatalf("want 1 page / 2 blocks, got %d pages", len(pages))
	}

	for i, pb := range pages[0].Blocks {
		if pb.Pos.W != GridColumns || pb.Pos.X != 0 || pb.Row != i {
			t.Fatalf("block %d: got X=%d W=%d Row=%d, want X=0 W=%d Row=%d",
				i, pb.Pos.X, pb.Pos.W, pb.Row, GridColumns, i)
		}
	}
}

func TestPackFlowProportionalFill(t *testing.T) {
	// 3 + 3 + 3 then a 4 wraps: the first row grows 3/3/3 -> 4/4/4, the 4 wraps
	// to a full-width second row.
	doc := resolvedDoc(t,
		chartBlock("a", 3, 4), chartBlock("b", 3, 4),
		chartBlock("c", 3, 4), chartBlock("d", 4, 4),
	)

	pages, err := Pack(doc, flowGeometry())
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	got := pages[0].Blocks
	if len(got) != 4 {
		t.Fatalf("want 4 blocks, got %d", len(got))
	}

	want := []struct{ x, w, row int }{
		{0, 4, 0}, {4, 4, 0}, {8, 4, 0}, {0, 12, 1},
	}
	for i, w := range want {
		if got[i].Pos.X != w.x || got[i].Pos.W != w.w || got[i].Row != w.row {
			t.Fatalf("block %d: got X=%d W=%d Row=%d, want X=%d W=%d Row=%d",
				i, got[i].Pos.X, got[i].Pos.W, got[i].Row, w.x, w.w, w.row)
		}
	}
}

func TestPackFlowPosYForcesNewRow(t *testing.T) {
	// 4 + 4 would share a row, but the second block's Pos.Y is higher.
	a := chartBlock("a", 4, 4)
	b := chartBlock("b", 4, 4)
	b.Pos.Y = 2

	pages, err := Pack(resolvedDoc(t, a, b), flowGeometry())
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	got := pages[0].Blocks
	if len(got) != 2 || got[0].Row != 0 || got[1].Row != 1 {
		t.Fatalf("Pos.Y bump must split the row, got rows %d/%d", got[0].Row, got[1].Row)
	}

	if got[0].Pos.W != GridColumns || got[1].Pos.W != GridColumns {
		t.Fatalf("each lone block should fill the width, got %d/%d", got[0].Pos.W, got[1].Pos.W)
	}
}

func TestPackFlowPaginatesByEstimatedHeight(t *testing.T) {
	blocks := make([]models.Block, 8)
	for i := range blocks {
		blocks[i] = chartBlock(string(rune('a'+i)), 12, 4)
	}

	pages, err := Pack(resolvedDoc(t, blocks...), flowGeometry())
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if len(pages) < 2 {
		t.Fatalf("8 full-width charts must paginate, got %d page(s)", len(pages))
	}

	total := 0
	for i, p := range pages {
		for _, pb := range p.Blocks {
			if pb.PageIndex != i {
				t.Fatalf("block PageIndex %d on page %d", pb.PageIndex, i)
			}

			total++
		}
	}

	if total != 8 {
		t.Fatalf("pages cover %d blocks, want 8", total)
	}
}

func TestPackFlowSplitsTallTable(t *testing.T) {
	rows := make([]map[string]any, 200)
	for i := range rows {
		rows[i] = map[string]any{"n": i}
	}

	doc, err := Resolve(tableLayout(nil, nil), map[string][]map[string]any{"d": rows})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	pages, err := Pack(doc, flowGeometry())
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

			nextOffset += pb.RowCount
			total += pb.RowCount
		}
	}

	if total != 200 {
		t.Fatalf("chunks cover %d rows, want 200", total)
	}
}

func TestPageGeometryForWorkArea(t *testing.T) {
	plain := PageGeometryFor(models.PageConfig{})

	if plain.Cols != GridColumns || plain.Rows != GridRows {
		t.Fatalf("grid stays %dx%d, got %dx%d", GridColumns, GridRows, plain.Cols, plain.Rows)
	}

	wantH := a4HeightMM - 2*DefaultPaddingMM
	if plain.HeightMM != wantH {
		t.Fatalf("plain work height = %d, want %d", plain.HeightMM, wantH)
	}

	if plain.RowHeightMM != float64(wantH)/float64(GridRows) {
		t.Fatalf("row height = %v, want %v", plain.RowHeightMM, float64(wantH)/float64(GridRows))
	}

	chrome := PageGeometryFor(models.PageConfig{
		Header: &models.PageSection{Text: "h"},
		Footer: &models.PageSection{HTML: "<b>f</b>"},
	})

	if got, want := chrome.HeightMM, wantH-HeaderBandMM-FooterBandMM; got != want {
		t.Fatalf("chrome work height = %d, want %d", got, want)
	}

	land := PageGeometryFor(models.PageConfig{Orientation: "landscape"})
	if land.WidthMM != a4HeightMM-2*DefaultPaddingMM {
		t.Fatalf("landscape work width = %d, want %d", land.WidthMM, a4HeightMM-2*DefaultPaddingMM)
	}
}
