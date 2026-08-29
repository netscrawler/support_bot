package layout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"support_bot/internal/models"
)

func TestLayoutFromOrder(t *testing.T) {
	got := LayoutFromOrder(map[string][]string{
		"countries": []string{"country:Страна", "count:Количество"},
		"daily":     []string{"date:Дата", "accepted:Принято"},
	})

	if got.Version != 1 {
		t.Fatalf("unexpected version: %d", got.Version)
	}

	if got.Page.Format != "A4" || got.Page.PaddingMM != DefaultPaddingMM {
		t.Fatalf("unexpected page defaults: %+v", got.Page)
	}

	if len(got.Blocks) != 2 {
		t.Fatalf("unexpected block count: %d", len(got.Blocks))
	}

	if got.Blocks[0].Dataset != "countries" {
		t.Fatalf("expected sorted block order, got %+v", got.Blocks)
	}

	if got.Blocks[0].Table == nil || len(got.Blocks[0].Table.Columns) != 2 {
		t.Fatalf("expected table columns, got %+v", got.Blocks[0])
	}

	if got.Blocks[0].Table.Columns[0].Title != "Страна" {
		t.Fatalf("unexpected column title: %+v", got.Blocks[0].Table.Columns[0])
	}
}

func TestLayoutFromTemplate(t *testing.T) {
	got := LayoutFromTemplate(&models.Template{
		Title:        "Example template",
		TemplateText: "Hello {{ .Name }}",
		Type:         "rich_text",
	})

	if len(got.Blocks) != 1 {
		t.Fatalf("unexpected block count: %d", len(got.Blocks))
	}

	block := got.Blocks[0]
	if block.Type != models.BlockTypeText || block.Text == nil {
		t.Fatalf("unexpected block: %+v", block)
	}

	if block.Text.Text != "Hello {{ .Name }}" {
		t.Fatalf("unexpected text content: %q", block.Text.Text)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		layout      models.Layout
		datasets    []string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid layout",
			layout: models.Layout{
				Version: 1,
				Blocks: []models.Block{
					{
						ID:      "daily-chart",
						Type:    models.BlockTypeChart,
						Dataset: "daily",
						Pos:     models.GridPosition{X: 0, Y: 0, W: 6, H: 4},
						Chart: &models.ChartBlock{
							Kind:   models.ChartKindLine,
							XField: "date",
							Series: []models.Series{{Field: "accepted", Title: "Принято"}},
						},
					},
				},
			},
			datasets: []string{"daily"},
		},
		{
			name: "unknown type",
			layout: models.Layout{
				Blocks: []models.Block{
					{
						ID:   "x",
						Type: "unknown",
					},
				},
			},
			wantErr:     true,
			errContains: "unknown type",
		},
		{
			name: "duplicate id",
			layout: models.Layout{
				Blocks: []models.Block{
					{ID: "x", Type: models.BlockTypeText, Text: &models.TextBlock{Text: "a"}},
					{ID: "x", Type: models.BlockTypeText, Text: &models.TextBlock{Text: "b"}},
				},
			},
			wantErr:     true,
			errContains: "duplicate id",
		},
		{
			name: "missing dataset",
			layout: models.Layout{
				Blocks: []models.Block{
					{
						ID:      "daily",
						Type:    models.BlockTypeTable,
						Dataset: "missing",
						Table: &models.TableBlock{
							Columns: []models.Column{{Field: "value"}},
						},
					},
				},
			},
			datasets:    []string{"daily"},
			wantErr:     true,
			errContains: "dataset",
		},
		{
			name: "pie with multiple series",
			layout: models.Layout{
				Blocks: []models.Block{
					{
						ID:      "pie",
						Type:    models.BlockTypeChart,
						Dataset: "daily",
						Pos:     models.GridPosition{X: 0, Y: 0, W: 6, H: 4},
						Chart: &models.ChartBlock{
							Kind:   models.ChartKindPie,
							XField: "status",
							Series: []models.Series{
								{Field: "accepted"},
								{Field: "rejected"},
							},
						},
					},
				},
			},
			datasets:    []string{"daily"},
			wantErr:     true,
			errContains: "pie chart supports exactly one series",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.layout, tt.datasets)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("unexpected error: %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLayoutJSONExample(t *testing.T) {
	path := filepath.Join("..", "models", "layout.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read layout json: %v", err)
	}

	var layout models.Layout
	if err := json.Unmarshal(data, &layout); err != nil {
		t.Fatalf("unmarshal layout json: %v", err)
	}

	if err := Validate(layout, []string{"daily"}); err != nil {
		t.Fatalf("validate layout json: %v", err)
	}
}
