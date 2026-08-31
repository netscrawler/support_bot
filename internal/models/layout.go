package models

import "strings"

type Layout struct {
	Version int        `json:"version"`
	Page    PageConfig `json:"page"`
	Blocks  []Block    `json:"blocks"`
}

type PageConfig struct {
	Format      string       `json:"format,omitempty"`
	Orientation string       `json:"orientation,omitempty"`
	PaddingMM   int          `json:"padding_mm,omitempty"`
	Header      *PageSection `json:"header,omitempty"`
	Footer      *PageSection `json:"footer,omitempty"`
}

type PageSection struct {
	Text string `json:"text,omitempty"`
	HTML string `json:"html,omitempty"`
}

type BlockType string

const (
	BlockTypeTable   BlockType = "table"
	BlockTypeChart   BlockType = "chart"
	BlockTypeMetric  BlockType = "metric"
	BlockTypeText    BlockType = "text"
	BlockTypeRawHTML BlockType = "raw_html"
)

type GridPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Block struct {
	ID      string       `json:"id"`
	Type    BlockType    `json:"type"`
	Dataset string       `json:"dataset,omitempty"`
	Pos     GridPosition `json:"pos"`

	Table   *TableBlock   `json:"table,omitempty"`
	Chart   *ChartBlock   `json:"chart,omitempty"`
	Metric  *MetricBlock  `json:"metric,omitempty"`
	Text    *TextBlock    `json:"text,omitempty"`
	RawHTML *RawHTMLBlock `json:"raw_html,omitempty"`
}

type ColumnAlign string

const (
	ColumnAlignLeft   ColumnAlign = "left"
	ColumnAlignCenter ColumnAlign = "center"
	ColumnAlignRight  ColumnAlign = "right"
)

type Column struct {
	Field  string      `json:"field"`
	Title  string      `json:"title,omitempty"`
	Width  int         `json:"width,omitempty"`
	Format string      `json:"format,omitempty"`
	Align  ColumnAlign `json:"align,omitempty"`
}

type TableSort struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc,omitempty"`
}

type TableBlock struct {
	// Title is an optional heading rendered above the table, matching
	// ChartBlock.Title.
	Title       string      `json:"title,omitempty"`
	Columns     []Column    `json:"columns,omitempty"`
	Sort        []TableSort `json:"sort,omitempty"`
	Limit       *int        `json:"limit,omitempty"`
	Conditional []CondRule  `json:"conditional,omitempty"`
}

// CondTarget selects what a CondRule paints when its expression matches a row.
type CondTarget string

const (
	CondTargetCell CondTarget = "cell"
	CondTargetRow  CondTarget = "row"
)

// CondStyle names a built-in colour preset for a CondRule: a semantic name or,
// when the meaning is just "tint it", a plain colour from the report palette.
type CondStyle string

const (
	// Semantic presets.
	CondStyleGood    CondStyle = "good"
	CondStyleWarning CondStyle = "warning"
	CondStyleBad     CondStyle = "bad"
	CondStyleMuted   CondStyle = "muted"

	// Plain colour presets — soft tints that stay inside the report style.
	CondStyleRed    CondStyle = "red"
	CondStyleOrange CondStyle = "orange"
	CondStyleAmber  CondStyle = "amber"
	CondStyleYellow CondStyle = "yellow"
	CondStyleGreen  CondStyle = "green"
	CondStyleTeal   CondStyle = "teal"
	CondStyleBlue   CondStyle = "blue"
	CondStyleIndigo CondStyle = "indigo"
	CondStylePurple CondStyle = "purple"
	CondStylePink   CondStyle = "pink"
)

// CondStyles lists every recognised preset name, in declaration order.
var CondStyles = []CondStyle{
	CondStyleGood, CondStyleWarning, CondStyleBad, CondStyleMuted,
	CondStyleRed, CondStyleOrange, CondStyleAmber, CondStyleYellow, CondStyleGreen,
	CondStyleTeal, CondStyleBlue, CondStyleIndigo, CondStylePurple, CondStylePink,
}

var condStyleSet = func() map[CondStyle]struct{} {
	m := make(map[CondStyle]struct{}, len(CondStyles))
	for _, s := range CondStyles {
		m[s] = struct{}{}
	}

	return m
}()

// Valid reports whether s is a recognised preset name. The empty string is not
// valid here; callers treat it as "no preset".
func (s CondStyle) Valid() bool {
	_, ok := condStyleSet[s]

	return ok
}

// CondRule is one conditional-formatting rule on a TableBlock. When the CEL
// expression When evaluates true for a row, Target decides whether the whole
// row or the named Columns are decorated. Style names a preset; BG, FG, Bold
// and Italic override it. Rules apply in declaration order, last match wins per
// cell, and a cell rule out-ranks a row rule on the same cell. The expression
// sees row (this row's fields), value (the current cell) and rows (every row in
// the block).
type CondRule struct {
	When    string     `json:"when"`
	Target  CondTarget `json:"target,omitempty"`
	Columns []string   `json:"columns,omitempty"`
	Style   CondStyle  `json:"style,omitempty"`
	BG      string     `json:"bg,omitempty"`
	FG      string     `json:"fg,omitempty"`
	Bold    bool       `json:"bold,omitempty"`
	Italic  bool       `json:"italic,omitempty"`
}

type ChartKind string

const (
	ChartKindLine ChartKind = "line"
	ChartKindBar  ChartKind = "bar"
	ChartKindPie  ChartKind = "pie"
)

type Series struct {
	Field string `json:"field"`
	Title string `json:"title,omitempty"`
	Color string `json:"color,omitempty"`
}

func (s Series) TitleOrField() string {
	if strings.TrimSpace(s.Title) != "" {
		return s.Title
	}
	return s.Field
}

type ChartBlock struct {
	Kind    ChartKind `json:"kind"`
	XField  string    `json:"x_field"`
	Series  []Series  `json:"series,omitempty"`
	Stacked bool      `json:"stacked,omitempty"`
	// Title is an optional heading rendered above the chart.
	Title string `json:"title,omitempty"`
	// Legend toggles the chart legend; nil leaves the renderer default.
	Legend *bool `json:"legend,omitempty"`
	// HeightPx is the physical canvas height in pixels; 0 uses the renderer
	// default. Distinct from the grid height in Block.Pos.H.
	HeightPx int `json:"height_px,omitempty"`
}

type MetricBlock struct {
	Field  string `json:"field"`
	Label  string `json:"label,omitempty"`
	Format string `json:"format,omitempty"`
}

// TextBlock renders Text as a Go template over the report dataset. In the html
// and pdf outputs it is executed with html/template (values are context
// escaped); in the text output it is executed with text/template. Use
// RawHTMLBlock to emit unescaped markup.
type TextBlock struct {
	Text string `json:"text"`
}

// RawHTMLBlock is emitted verbatim in the html and pdf outputs and as plain
// text in xlsx. The author is responsible for the markup being safe.
type RawHTMLBlock struct {
	HTML string `json:"html"`
}
