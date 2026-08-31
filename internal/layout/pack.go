package layout

import (
	"fmt"
	"math"

	"support_bot/internal/models"
)

// Table auto-height heuristics, shared by Resolve and Pack.
// A grid row holds RowsPerGridRow data rows; HeaderRows grid rows are reserved
// for the table header.
const (
	RowsPerGridRow = 6
	HeaderRows     = 1
)

// PageGeometry describes the target surface for Pack.
//
//   - Rows == 0            => a single unbounded page, no pagination, no
//     splitting; each block is placed at its PreferredSize.
//   - RespectAuthoredPos   => a single page; each block is placed at its authored
//     Block.Pos verbatim (used by the spreadsheet renderer).
//   - Flow                 => blocks flow left-to-right in declaration order; a
//     block that does not fit the remaining columns wraps to a new row and the
//     blocks left behind grow proportionally to fill it. Pagination is driven by
//     estimated millimetre heights against HeightMM, so WidthMM / HeightMM /
//     RowHeightMM must be set.
//   - otherwise            => first-fit row-major packing across Cols x Rows
//     pages, splitting Splittable blocks across page boundaries.
//
// WidthMM / HeightMM / RowHeightMM carry the physical work-area for renderers
// that need millimetre sizing; the non-Flow modes of Pack only use Cols, Rows
// and RespectAuthoredPos.
type PageGeometry struct {
	Cols               int
	Rows               int
	WidthMM            int
	HeightMM           int
	RowHeightMM        float64
	RespectAuthoredPos bool
	Flow               bool
}

// PlacedBlock is a ResolvedBlock with a final grid position and the window of
// Rows that belongs on this page. For non-split blocks RowOffset is 0 and
// RowCount is len(Rows).
//
// Row is the 0-based flow row this block sits on within its page (Flow mode
// only; 0 for every other mode).
type PlacedBlock struct {
	ResolvedBlock

	Pos       models.GridPosition
	PageIndex int
	Row       int
	RowOffset int
	RowCount  int
}

// Page is an ordered set of placed blocks.
type Page struct {
	Index  int
	Blocks []PlacedBlock
}

// Pack assigns every block in doc a concrete grid position and page index.
func Pack(doc ResolvedDocument, geom PageGeometry) ([]Page, error) {
	cols := geom.Cols
	if cols <= 0 {
		cols = GridColumns
	}

	switch {
	case geom.RespectAuthoredPos:
		return singlePage(doc, authoredPos), nil
	case geom.Rows <= 0:
		return singlePage(doc, preferredPos), nil
	case geom.Flow:
		return packFlow(doc, cols, geom), nil
	default:
		return packGrid(doc, cols, geom.Rows)
	}
}

func authoredPos(rb ResolvedBlock) models.GridPosition  { return rb.Block.Pos }
func preferredPos(rb ResolvedBlock) models.GridPosition { return rb.PreferredSize }

func singlePage(doc ResolvedDocument, pos func(ResolvedBlock) models.GridPosition) []Page {
	page := Page{Index: 0, Blocks: make([]PlacedBlock, 0, len(doc.Blocks))}

	for _, rb := range doc.Blocks {
		page.Blocks = append(page.Blocks, PlacedBlock{
			ResolvedBlock: rb,
			Pos:           pos(rb),
			PageIndex:     0,
			RowOffset:     0,
			RowCount:      len(rb.Rows),
		})
	}

	return []Page{page}
}

// workBlock is the mutable placement state for one block (or the not-yet-placed
// remainder of a split table).
type workBlock struct {
	rb         ResolvedBlock
	w          int
	h          int
	offset     int
	count      int
	splittable bool
}

func packGrid(doc ResolvedDocument, cols, rows int) ([]Page, error) {
	queue := make([]workBlock, 0, len(doc.Blocks))
	for _, rb := range doc.Blocks {
		queue = append(queue, workBlock{
			rb:         rb,
			w:          rb.PreferredSize.W,
			h:          rb.PreferredSize.H,
			offset:     0,
			count:      len(rb.Rows),
			splittable: rb.Splittable,
		})
	}

	pages := []Page{{Index: 0}}
	grid := newGrid(cols, rows)

	for i := 0; i < len(queue); {
		wb := queue[i]

		if err := validateWorkBlock(wb, cols, rows); err != nil {
			return nil, err
		}

		rem, ok := tryPlace(grid, &pages[len(pages)-1], wb, cols, rows)
		if !ok {
			grid = newGrid(cols, rows)
			pages = append(pages, Page{Index: len(pages)})
			rem, ok = tryPlace(grid, &pages[len(pages)-1], wb, cols, rows)
		}

		if !ok {
			if !wb.splittable {
				return nil, fmt.Errorf(
					"%s: block too large for page: %dx%d > %dx%d",
					wb.rb.Block.Type, wb.w, wb.h, cols, rows,
				)
			}

			return nil, fmt.Errorf("%s: cannot place block on new page", wb.rb.Block.Type)
		}

		if rem != nil {
			queue[i] = *rem

			continue
		}

		i++
	}

	return pages, nil
}

func validateWorkBlock(wb workBlock, cols, rows int) error {
	if wb.w <= 0 || wb.w > cols {
		return fmt.Errorf(`%s: option "w" must be between 1 and %d`, wb.rb.Block.Type, cols)
	}

	if !wb.splittable && (wb.h <= 0 || wb.h > rows) {
		return fmt.Errorf(`%s: option "h" must be between 1 and %d`, wb.rb.Block.Type, rows)
	}

	return nil
}

// tryPlace scans the grid row-major and places wb (or a leading chunk of it).
// It returns the unplaced remainder, if any, and whether anything was placed.
func tryPlace(g *grid, page *Page, wb workBlock, cols, rows int) (*workBlock, bool) {
	for y := range rows {
		for x := 0; x <= cols-wb.w; x++ {
			if wb.h > 0 && g.canPlace(x, y, wb.w, wb.h) {
				g.place(x, y, wb.w, wb.h)
				appendPlaced(page, wb, x, y, wb.h, wb.offset, wb.count)

				return nil, true
			}

			if !wb.splittable || wb.count <= 0 {
				continue
			}

			avail := g.availableHeight(x, y, wb.w)
			if avail <= HeaderRows {
				continue
			}

			capacity := min((avail-HeaderRows)*RowsPerGridRow, wb.count)
			chunkH := min(HeaderRows+ceilDiv(capacity, RowsPerGridRow), avail)

			g.place(x, y, wb.w, chunkH)
			appendPlaced(page, wb, x, y, chunkH, wb.offset, capacity)

			if capacity >= wb.count {
				return nil, true
			}

			remRows := wb.count - capacity

			return &workBlock{
				rb:         wb.rb,
				w:          wb.w,
				h:          HeaderRows + ceilDiv(remRows, RowsPerGridRow),
				offset:     wb.offset + capacity,
				count:      remRows,
				splittable: true,
			}, true
		}
	}

	return nil, false
}

func appendPlaced(page *Page, wb workBlock, x, y, h, rowOffset, rowCount int) {
	page.Blocks = append(page.Blocks, PlacedBlock{
		ResolvedBlock: wb.rb,
		Pos:           models.GridPosition{X: x, Y: y, W: wb.w, H: h},
		PageIndex:     page.Index,
		RowOffset:     rowOffset,
		RowCount:      rowCount,
	})
}

type grid struct {
	cols     int
	rows     int
	occupied [][]bool
}

func newGrid(cols, rows int) *grid {
	occ := make([][]bool, rows)
	for i := range occ {
		occ[i] = make([]bool, cols)
	}

	return &grid{cols: cols, rows: rows, occupied: occ}
}

func (g *grid) canPlace(x, y, w, h int) bool {
	if x < 0 || y < 0 || w <= 0 || h <= 0 {
		return false
	}

	if x+w > g.cols || y+h > g.rows {
		return false
	}

	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if g.occupied[yy][xx] {
				return false
			}
		}
	}

	return true
}

func (g *grid) place(x, y, w, h int) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			g.occupied[yy][xx] = true
		}
	}
}

func (g *grid) availableHeight(x, y, w int) int {
	free := 0

	for yy := y; yy < g.rows; yy++ {
		rowFree := true

		for xx := x; xx < x+w; xx++ {
			if g.occupied[yy][xx] {
				rowFree = false

				break
			}
		}

		if !rowFree {
			break
		}

		free++
	}

	return free
}

func ceilDiv(a, b int) int {
	if b <= 0 {
		return a
	}

	return (a + b - 1) / b
}

// Millimetre estimates for the flow packer. Server-side rendering cannot measure
// content, so these are set from the rendered px height of each block kind in
// headless Chromium (see TestFixtureLayoutFitsA4), plus a slim margin: they must
// stay slightly *over* reality, since an over-filled .report-page div spills
// past its A4 @page box and doubles the PDF page count. Measured at 96 dpi.
const (
	pxPerMM = 3.779527559

	metricBlockMM    = 26.0 // measured 23mm
	chartChromeMM    = 19.0 // card padding + title band; measured 17mm
	chartMinBodyMM   = 50.0
	defaultChartPx   = 240
	tableHeaderMM    = 12.0 // header row + card padding; measured 9.5mm
	tableRowMM       = 9.0  // measured 8.7mm
	textDefaultMM    = 17.0 // measured 14mm
	rawHTMLDefaultMM = 15.0
	rowGapMM         = 5.0 // .report-grid gap is 3.7mm

	// breakSlackMM lets a row overflow the work-area by a hair rather than
	// opening a near-empty page for a few millimetres of rounding.
	breakSlackMM = 3.0
)

// flowItem is one block on a flow row, with its final column span and (for a
// split table) the window of Rows that belongs on this row.
type flowItem struct {
	rb     ResolvedBlock
	x      int
	w      int
	offset int
	count  int
}

type flowRow struct {
	items    []flowItem
	heightMM float64
}

// packFlow lays blocks out in declaration order: a block that does not fit the
// remaining columns of the current row wraps to a new one, and the blocks left
// behind grow proportionally to fill the width. Pages break when the estimated
// millimetre height of the next row would overflow geom.HeightMM; a splittable
// table taller than a page is split by row count.
func packFlow(doc ResolvedDocument, cols int, geom PageGeometry) []Page {
	availMM := float64(geom.HeightMM)
	if availMM <= 0 {
		availMM = float64(a4HeightMM - 2*DefaultPaddingMM)
	}

	rowHeightMM := geom.RowHeightMM
	if rowHeightMM <= 0 {
		rowHeightMM = availMM / float64(GridRows)
	}

	queue := formFlowRows(doc.Blocks, cols)

	var pages []Page

	page := Page{Index: 0}
	usedMM := 0.0
	rowIdx := 0

	newPage := func() {
		pages = append(pages, page)
		page = Page{Index: len(pages)}
		usedMM = 0
		rowIdx = 0
	}

	appendRow := func(r flowRow) {
		span := gridSpan(r.heightMM, rowHeightMM)
		for _, it := range r.items {
			page.Blocks = append(page.Blocks, PlacedBlock{
				ResolvedBlock: it.rb,
				Pos: models.GridPosition{
					X: it.x, Y: rowIdx, W: it.w, H: span,
				},
				PageIndex: page.Index,
				Row:       rowIdx,
				RowOffset: it.offset,
				RowCount:  it.count,
			})
		}

		usedMM += r.heightMM
		rowIdx++
	}

	for qi := 0; qi < len(queue); qi++ {
		r := queue[qi]
		r.heightMM = flowRowHeightMM(r, rowHeightMM)

		splittable := len(r.items) == 1 && r.items[0].rb.Splittable

		if usedMM > 0 && usedMM+r.heightMM > availMM+breakSlackMM &&
			(!splittable || r.heightMM <= availMM) {
			newPage()
		}

		if splittable && r.heightMM > availMM-usedMM {
			it := r.items[0]

			avail := availMM - usedMM
			fit := int((avail - tableHeaderMM - rowGapMM) / tableRowMM)

			if fit < 1 && usedMM > 0 {
				newPage()

				avail = availMM
				fit = int((avail - tableHeaderMM - rowGapMM) / tableRowMM)
			}

			if fit < 1 {
				fit = 1
			}

			if fit < it.count {
				chunk := flowRow{items: []flowItem{{
					rb: it.rb, x: it.x, w: it.w, offset: it.offset, count: fit,
				}}}
				chunk.heightMM = flowRowHeightMM(chunk, rowHeightMM)
				appendRow(chunk)

				rem := flowRow{items: []flowItem{{
					rb: it.rb, x: 0, w: cols, offset: it.offset + fit, count: it.count - fit,
				}}}
				tail := append([]flowRow{rem}, queue[qi+1:]...)
				queue = append(queue[:qi+1], tail...)

				newPage()

				continue
			}
		}

		appendRow(r)
	}

	pages = append(pages, page)

	return pages
}

// formFlowRows groups blocks into rows: a new row starts when the next block
// would overflow the column count, when its authored Pos.Y is greater than the
// previous block's (a grouping hint), or when either block is a table (a table
// always occupies its own full-width row so a narrow column can never clip it).
// Each closed row is grown to the full column count in fillRow.
func formFlowRows(blocks []ResolvedBlock, cols int) []flowRow {
	var (
		rows  []flowRow
		cur   []flowItem
		curW  int
		prevY int
		hasY  bool
	)

	flush := func() {
		if len(cur) == 0 {
			return
		}

		rows = append(rows, flowRow{items: fillRow(cur, cols)})
		cur = nil
		curW = 0
	}

	for _, rb := range blocks {
		w := rb.PreferredSize.W
		if w <= 0 {
			w = cols
		}

		if w > cols {
			w = cols
		}

		y := rb.Block.Pos.Y
		isTable := rb.Block.Type == models.BlockTypeTable

		lastIsTable := len(cur) > 0 &&
			cur[len(cur)-1].rb.Block.Type == models.BlockTypeTable

		if len(cur) > 0 &&
			(isTable || lastIsTable || curW+w > cols || (hasY && y > prevY)) {
			flush()
		}

		cur = append(cur, flowItem{rb: rb, w: w, count: len(rb.Rows)})
		curW += w
		prevY = y
		hasY = true

		if isTable || curW >= cols {
			flush()
		}
	}

	flush()

	return rows
}

// fillRow distributes the leftover columns of a short row across its blocks in
// proportion to their widths, handing any remainder to the last block, then
// assigns each block its x offset. A single block on a row becomes full width.
func fillRow(items []flowItem, cols int) []flowItem {
	sum := 0
	for _, it := range items {
		sum += it.w
	}

	if sum < cols && sum > 0 {
		extra := cols - sum
		added := 0

		for i := range items {
			add := extra * items[i].w / sum
			items[i].w += add
			added += add
		}

		items[len(items)-1].w += extra - added
	}

	x := 0
	for i := range items {
		items[i].x = x
		x += items[i].w
	}

	return items
}

func flowRowHeightMM(r flowRow, rowHeightMM float64) float64 {
	maxH := 0.0

	for _, it := range r.items {
		if h := blockHeightMM(it.rb, it.count, rowHeightMM); h > maxH {
			maxH = h
		}
	}

	return maxH + rowGapMM
}

func blockHeightMM(rb ResolvedBlock, rowCount int, rowHeightMM float64) float64 {
	switch rb.Block.Type {
	case models.BlockTypeMetric:
		return metricBlockMM
	case models.BlockTypeChart:
		px := defaultChartPx
		if rb.Block.Chart != nil && rb.Block.Chart.HeightPx > 0 {
			px = rb.Block.Chart.HeightPx
		}

		body := math.Max(float64(px)/pxPerMM, chartMinBodyMM)

		return chartChromeMM + body
	case models.BlockTypeTable:
		return tableHeaderMM + float64(rowCount)*tableRowMM
	case models.BlockTypeText:
		if rb.Block.Pos.H > 0 {
			return float64(rb.Block.Pos.H) * rowHeightMM
		}

		return textDefaultMM
	case models.BlockTypeRawHTML:
		if rb.Block.Pos.H > 0 {
			return float64(rb.Block.Pos.H) * rowHeightMM
		}

		return rawHTMLDefaultMM
	default:
		return rowHeightMM
	}
}

func gridSpan(heightMM, rowHeightMM float64) int {
	if rowHeightMM <= 0 {
		return 1
	}

	if s := int(math.Ceil(heightMM / rowHeightMM)); s > 1 {
		return s
	}

	return 1
}
