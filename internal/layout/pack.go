package layout

import (
	"fmt"

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
//   - otherwise            => first-fit row-major packing across Cols x Rows
//     pages, splitting Splittable blocks across page boundaries.
//
// WidthMM / HeightMM / RowHeightMM carry the physical work-area for renderers
// that need millimetre sizing; Pack itself only uses Cols, Rows and
// RespectAuthoredPos.
type PageGeometry struct {
	Cols               int
	Rows               int
	WidthMM            int
	HeightMM           int
	RowHeightMM        float64
	RespectAuthoredPos bool
}

// PlacedBlock is a ResolvedBlock with a final grid position and the window of
// Rows that belongs on this page. For non-split blocks RowOffset is 0 and
// RowCount is len(Rows).
type PlacedBlock struct {
	ResolvedBlock

	Pos       models.GridPosition
	PageIndex int
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
