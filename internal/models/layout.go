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
	Columns []Column    `json:"columns,omitempty"`
	Sort    []TableSort `json:"sort,omitempty"`
	Limit   *int        `json:"limit,omitempty"`
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
}

type MetricBlock struct {
	Field  string `json:"field"`
	Label  string `json:"label,omitempty"`
	Format string `json:"format,omitempty"`
}

type TextBlock struct {
	Text string `json:"text"`
}

type RawHTMLBlock struct {
	HTML string `json:"html"`
}
