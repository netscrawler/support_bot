package html

import (
	"strings"
	"testing"
)

func TestParseSeries(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []seriesDefinition
		wantErr bool
	}{
		{
			name: "field and label",
			input: []string{
				"accepted:Принято",
				"rejected:Отклонено",
			},
			want: []seriesDefinition{
				{
					Field: "accepted",
					Label: "Принято",
				},
				{
					Field: "rejected",
					Label: "Отклонено",
				},
			},
		},
		{
			name: "field only",
			input: []string{
				"accepted",
			},
			want: []seriesDefinition{
				{
					Field: "accepted",
					Label: "accepted",
				},
			},
		},
		{
			name: "colon in label",
			input: []string{
				"amount:Сумма: RUB",
			},
			want: []seriesDefinition{
				{
					Field: "amount",
					Label: "Сумма: RUB",
				},
			},
		},
		{
			name:    "empty",
			input:   []string{""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSeries(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf(
					"length mismatch: got %d want %d",
					len(got),
					len(tt.want),
				)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf(
						"series[%d]: got %+v want %+v",
						i,
						got[i],
						tt.want[i],
					)
				}
			}
		})
	}
}

func TestLineChart(t *testing.T) {
	rows := []map[string]any{
		{
			"date":     "2026-08-01",
			"accepted": 120,
			"rejected": 10,
		},
		{
			"date":     "2026-08-02",
			"accepted": 140,
			"rejected": 12,
		},
	}

	result, err := newChart(
		"line",
		rows,
		"date",
		[]string{"accepted:Принято", "rejected:Отклонено"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := string(result)

	for _, expected := range []string{
		"report-chart-",
		"2026-08-01",
		"2026-08-02",
		"Принято",
		"Отклонено",
		"120",
		"140",
		"10",
		"12",
		`"type":"line"`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf(
				"expected HTML to contain %q\n%s",
				expected,
				html,
			)
		}
	}
}

func TestBarChart(t *testing.T) {
	rows := []map[string]any{
		{
			"country": "UZ",
			"count":   500,
		},
		{
			"country": "KG",
			"count":   300,
		},
	}

	result, err := newChart(
		"bar",
		rows,
		"country",
		[]string{"count:Количество"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := string(result)

	if !strings.Contains(html, `"type":"bar"`) {
		t.Fatalf("expected bar chart:\n%s", html)
	}
}

func TestPieChart(t *testing.T) {
	rows := []map[string]any{
		{
			"status": "Accepted",
			"count":  100,
		},
		{
			"status": "Rejected",
			"count":  20,
		},
	}

	result, err := newChart(
		"pie",
		rows,
		"status",
		[]string{"count:Количество"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := string(result)

	if !strings.Contains(html, `"type":"pie"`) {
		t.Fatalf("expected pie chart:\n%s", html)
	}
}

func TestPieChartRejectsMultipleSeries(t *testing.T) {
	_, err := newChart(
		"pie",
		[]map[string]any{
			{
				"status":   "Accepted",
				"accepted": 100,
				"rejected": 10,
			},
		},
		"status",
		[]string{"accepted:Принято", "rejected:Отклонено"},
		nil,
	)

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChartMissingField(t *testing.T) {
	_, err := newChart(
		"line",
		[]map[string]any{
			{
				"date": "2026-08-01",
			},
		},
		"date",
		[]string{"count:Количество"},
		nil,
	)

	if err == nil {
		t.Fatal("expected error")
	}
}
