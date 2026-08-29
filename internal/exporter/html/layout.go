package html

import "fmt"

// Layout and supporting types for report blocks on a logical grid.
// The implementation below keeps the original first-fit row-major placement
// and supports splitting of splittable blocks (tables). The code is refactored
// for clarity: helper functions and clear control flow.

// Default logical grid dimensions: 12 columns like common dashboard grids,
// 12 rows per A4 page.
const (
	GridColumns = 12
	GridRows    = 12
)

// BlockSize defines logical grid dimensions for a block.
type BlockSize struct {
	W int
	H int
}

// GridPosition is a 0-based X/Y position in the grid.
type GridPosition struct {
	X int
	Y int
}

// ReportBlock is a minimal representation of a block coming from templates.
// Content may contain rendered HTML; Size describes logical grid size.
// Splittable marks blocks that can be split across pages (tables).
type ReportBlock struct {
	Type           string
	Content        string
	Size           BlockSize
	ExplicitWidth  bool
	ExplicitHeight bool
	Splittable     bool

	// Table specific data (optional). If set, layout engine can split table rows.
	TableHeaders []string
	TableRows    [][]string
}

// PositionedBlock is a ReportBlock with computed grid position and page index.
type PositionedBlock struct {
	ReportBlock
	Position GridPosition
	Page     int
}

// Page holds positioned blocks for one page.
type Page struct {
	Blocks []PositionedBlock
}

// Grid represents occupancy of a page.
type Grid struct {
	Columns  int
	Rows     int
	occupied [][]bool
}

// NewGrid constructs a new empty grid.
func NewGrid(columns, rows int) *Grid {
	occ := make([][]bool, rows)
	for i := range occ {
		occ[i] = make([]bool, columns)
	}
	return &Grid{Columns: columns, Rows: rows, occupied: occ}
}

// CanPlace reports whether rectangle (x,y,w,h) fits inside the grid and is free.
func (g *Grid) CanPlace(x, y, w, h int) bool {
	if x < 0 || y < 0 || w <= 0 || h <= 0 {
		return false
	}
	if x+w > g.Columns || y+h > g.Rows {
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

// Place marks a rectangle (x,y,w,h) as occupied.
func (g *Grid) Place(x, y, w, h int) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			g.occupied[yy][xx] = true
		}
	}
}

// AvailableHeight returns how many consecutive free grid rows are available
// starting from (x,y) for width w.
func (g *Grid) AvailableHeight(x, y, w int) int {
	free := 0
	for yy := y; yy < g.Rows; yy++ {
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

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// LayoutEngine places ReportBlocks into pages.
type LayoutEngine interface {
	Layout(blocks []ReportBlock) ([]Page, error)
}

// SimpleLayoutEngine implements a row-major first-fit placement across pages.
type SimpleLayoutEngine struct {
	Columns int
	Rows    int
}

// NewSimpleLayoutEngine creates engine with given grid size.
func NewSimpleLayoutEngine(columns, rows int) *SimpleLayoutEngine {
	return &SimpleLayoutEngine{Columns: columns, Rows: rows}
}

// Layout places blocks using first-fit row-major algorithm with optional splitting
// for splittable blocks (tables). The function returns pages with positioned blocks.
func (e *SimpleLayoutEngine) Layout(blocks []ReportBlock) ([]Page, error) {
	if e.Columns <= 0 || e.Rows <= 0 {
		return nil, nil
	}

	pages := []Page{{}}
	grid := NewGrid(e.Columns, e.Rows)

	// Use index-based loop because splittable blocks may replace current entry with remainder.
	for i := 0; i < len(blocks); {
		b := blocks[i]

		// validate width
		if b.Size.W <= 0 || b.Size.W > e.Columns {
			return nil, fmt.Errorf("%s: option \"w\" must be between 1 and %d", b.Type, e.Columns)
		}

		// validate height for non-splittable blocks
		if !b.Splittable && (b.Size.H <= 0 || b.Size.H > e.Rows) {
			return nil, fmt.Errorf("%s: option \"h\" must be between 1 and %d", b.Type, e.Rows)
		}

		// attempt to place on current page
		placed, remainder, err := e.tryPlaceOnGrid(grid, &pages[len(pages)-1], blocks, i)
		if err != nil {
			return nil, err
		}
		if placed {
			if remainder {
				// block was split and remainder stored back into blocks[i]; process it next
				continue
			}
			// fully placed, advance to next block
			i++
			continue
		}

		// not placed on current page -> start a new page and try again
		grid = NewGrid(e.Columns, e.Rows)
		pages = append(pages, Page{})

		// attempt placement on new page
		placed, remainder, err = e.tryPlaceOnGrid(grid, &pages[len(pages)-1], blocks, i)
		if err != nil {
			return nil, err
		}
		if placed {
			if remainder {
				continue // remainder remains at blocks[i]
			}
			i++
			continue
		}

		// cannot place even on empty page
		if !b.Splittable {
			return nil, fmt.Errorf(
				"%s: block too large for page: %dx%d > %dx%d",
				b.Type,
				b.Size.W,
				b.Size.H,
				e.Columns,
				e.Rows,
			)
		}

		// if splittable but nothing could be placed (shouldn't normally happen), error
		return nil, fmt.Errorf("%s: cannot place block on new page", b.Type)
	}

	// fix Page index values for PositionedBlock.Page fields
	for pi := range pages {
		for bi := range pages[pi].Blocks {
			pages[pi].Blocks[bi].Page = pi
		}
	}

	return pages, nil
}

// tryPlaceOnGrid attempts to place block b on the provided grid/page.
// It either places the full block or, if splittable, places a chunk and
// updates blocks[idx] with the remainder (returns remainderPlaced=true).
func (e *SimpleLayoutEngine) tryPlaceOnGrid(
	grid *Grid,
	page *Page,
	blocks []ReportBlock,
	idx int,
) (placed bool, remainderPlaced bool, err error) {
	b := blocks[idx]
	w := b.Size.W
	h := b.Size.H

	for y := 0; y < e.Rows; y++ {
		for x := 0; x <= e.Columns-w; x++ {
			// try full placement
			if h > 0 && grid.CanPlace(x, y, w, h) {
				placedBlock := b
				// Tables are collected with empty Content and rendered during layout.
				// If a table fits entirely on page, render full content here.
				if placedBlock.Type == "table" && placedBlock.Content == "" {
					placedBlock.Content = renderTableChunk(
						placedBlock.TableHeaders,
						placedBlock.TableRows,
					)
				}
				grid.Place(x, y, w, h)
				page.Blocks = append(
					page.Blocks,
					PositionedBlock{
						ReportBlock: placedBlock,
						Position:    GridPosition{X: x, Y: y},
						Page:        len(page.Blocks), /* placeholder, corrected by caller */
					},
				)
				return true, false, nil
			}

			// try to place a chunk if block is splittable and carries table rows
			if b.Splittable && len(b.TableRows) > 0 {
				avail := grid.AvailableHeight(x, y, w)
				if avail <= 0 {
					continue
				}

				// capacity = number of data rows that fit into avail rows
				capacity := (avail - headerRowsCount) * rowsPerGridRow
				if capacity <= 0 {
					continue
				}

				if capacity > len(b.TableRows) {
					capacity = len(b.TableRows)
				}

				// compute grid height for chosen chunk
				chunkGridH := min(headerRowsCount+(capacity+rowsPerGridRow-1)/rowsPerGridRow, avail)

				// prepare placed chunk
				chunkRows := b.TableRows[:capacity]
				placedBlock := b
				placedBlock.TableRows = chunkRows
				placedBlock.Size.H = chunkGridH
				placedBlock.Content = renderTableChunk(b.TableHeaders, chunkRows)

				grid.Place(x, y, w, chunkGridH)
				page.Blocks = append(
					page.Blocks,
					PositionedBlock{
						ReportBlock: placedBlock,
						Position:    GridPosition{X: x, Y: y},
						Page:        len(page.Blocks),
					},
				)

				// write remainder into blocks[idx]
				remainderRows := b.TableRows[capacity:]
				remainderH := headerRowsCount + (len(remainderRows)+rowsPerGridRow-1)/rowsPerGridRow
				remainder := b
				remainder.TableRows = remainderRows
				remainder.Size.H = remainderH
				remainder.Content = ""
				blocks[idx] = remainder

				return true, true, nil
			}
		}
	}

	return false, false, nil
}
