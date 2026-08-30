package layout

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"support_bot/internal/models"
)

// defaultAutoTextRows is the grid height reserved for text / raw_html blocks
// that declare no explicit height. Server-side rendering cannot measure text,
// so a small fixed reservation is used.
const defaultAutoTextRows = 2

// ResolvedBlock is a layout block bound to a private copy of its dataset rows,
// with a computed preferred grid size. Rows is nil for blocks that carry no
// tabular data (text, raw_html).
type ResolvedBlock struct {
	Block         models.Block
	Rows          []map[string]any
	PreferredSize models.GridPosition
	Splittable    bool
}

// ResolvedDocument is a normalized layout with every block resolved against a
// dataset. It is the input to Pack and to every renderer.
type ResolvedDocument struct {
	Page   models.PageConfig
	Blocks []ResolvedBlock
}

// Resolve normalizes the layout, binds each block to a copy of its dataset
// rows, applies table Sort and Limit, and computes the preferred grid size for
// every block. The input data map and its slices are never mutated.
func Resolve(l models.Layout, data map[string][]map[string]any) (ResolvedDocument, error) {
	normalized := Normalize(l)

	doc := ResolvedDocument{
		Page:   normalized.Page,
		Blocks: make([]ResolvedBlock, 0, len(normalized.Blocks)),
	}

	for i := range normalized.Blocks {
		block := normalized.Blocks[i]

		var rows []map[string]any
		if block.Dataset != "" {
			if src, ok := data[block.Dataset]; ok {
				rows = slices.Clone(src)
			}
		}

		doc.Blocks = append(doc.Blocks, resolveBlock(block, rows))
	}

	return doc, nil
}

// ResolvedBlockFromRows resolves a single block whose rows are already bound
// positionally (the HTML template escape-hatch), without a dataset lookup.
func ResolvedBlockFromRows(block models.Block, rows []map[string]any) ResolvedBlock {
	normalized := Normalize(models.Layout{Blocks: []models.Block{block}}).Blocks[0]

	return resolveBlock(normalized, slices.Clone(rows))
}

func resolveBlock(block models.Block, rows []map[string]any) ResolvedBlock {
	resolved := ResolvedBlock{
		Block:      block,
		Rows:       rows,
		Splittable: block.Type == models.BlockTypeTable,
	}

	if block.Type == models.BlockTypeTable && block.Table != nil {
		if len(block.Table.Sort) > 0 {
			sortRows(resolved.Rows, block.Table.Sort)
		}

		if block.Table.Limit != nil {
			limit := *block.Table.Limit
			if limit >= 0 && limit < len(resolved.Rows) {
				resolved.Rows = resolved.Rows[:limit]
			}
		}
	}

	resolved.PreferredSize = preferredSize(block, len(resolved.Rows))

	return resolved
}

func preferredSize(block models.Block, rowCount int) models.GridPosition {
	size := models.GridPosition{
		X: block.Pos.X,
		Y: block.Pos.Y,
		W: block.Pos.W,
		H: block.Pos.H,
	}

	if size.W <= 0 {
		size.W = GridColumns
	}

	if size.H > 0 {
		return size
	}

	switch block.Type {
	case models.BlockTypeTable:
		size.H = max(HeaderRows+ceilDiv(rowCount, RowsPerGridRow), 1)
	case models.BlockTypeText, models.BlockTypeRawHTML:
		size.H = defaultAutoTextRows
	default:
		if def, ok := registry[block.Type]; ok && def.DefaultSize.H > 0 {
			size.H = def.DefaultSize.H
		} else {
			size.H = 1
		}
	}

	return size
}

func sortRows(rows []map[string]any, sorts []models.TableSort) {
	slices.SortStableFunc(rows, func(a, b map[string]any) int {
		for _, s := range sorts {
			cmp := compareValues(a[s.Field], b[s.Field])
			if cmp == 0 {
				continue
			}

			if s.Desc {
				return -cmp
			}

			return cmp
		}

		return 0
	})
}

func compareValues(a, b any) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	}

	if af, ok := toFloat(a); ok {
		if bf, ok := toFloat(b); ok {
			return cmpFloat(af, bf)
		}
	}

	if at, ok := a.(time.Time); ok {
		if bt, ok := b.(time.Time); ok {
			return cmpTime(at, bt)
		}
	}

	if ab, ok := a.(bool); ok {
		if bb, ok := b.(bool); ok {
			return cmpBool(ab, bb)
		}
	}

	return strings.Compare(toString(a), toString(b))
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()

		return f, err == nil
	default:
		return 0, false
	}
}

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

func cmpBool(a, b bool) int {
	switch {
	case !a && b:
		return -1
	case a && !b:
		return 1
	default:
		return 0
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	return fmt.Sprint(v)
}
