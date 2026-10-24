package httpcoll

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// testXLSX собирает книгу с листами Summary и Data.
func testXLSX(t *testing.T) string {
	t.Helper()

	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })

	require.NoError(t, f.SetSheetName("Sheet1", "Summary"))
	require.NoError(t, f.SetCellValue("Summary", "A1", "total"))

	_, err := f.NewSheet("Data")
	require.NoError(t, err)
	require.NoError(t, f.SetSheetRow("Data", "A1", &[]any{"date", "count"}))
	require.NoError(t, f.SetSheetRow("Data", "A2", &[]any{"2026-10-01", "5"}))
	require.NoError(t, f.SetSheetRow("Data", "A4", &[]any{"2026-10-02"}))

	var buf bytes.Buffer

	require.NoError(t, f.Write(&buf))

	return buf.String()
}

func TestDecode(t *testing.T) {
	t.Parallel()

	const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

	xlsx := testXLSX(t)

	tests := []struct {
		name                                    string
		format, ctype, root, delim, sheet, body string
		want                                    []map[string]any
		wantErr                                 string
	}{
		{
			name: "json root and flatten", ctype: "application/json",
			root: "data.items",
			body: `{"data":{"items":[{"id":1,"user":{"name":"a"}}]}}`,
			want: []map[string]any{{"id": float64(1), "user.name": "a"}},
		},
		{
			name:  "json scalars",
			ctype: "application/json; charset=utf-8",
			body:  `[1,2]`,
			want: []map[string]any{
				{"value": float64(1)},
				{"value": float64(2)},
			},
		},
		{
			name: "json object", ctype: "application/json", body: `{"a":1}`,
			want: []map[string]any{{"a": float64(1)}},
		},
		{
			name: "json missing root", ctype: "application/json",
			root: "data.rows", body: `{"data":{"x":1}}`, wantErr: "data.rows",
		},
		{
			name: "json root through scalar", ctype: "application/json",
			root: "a.b", body: `{"a":1}`, wantErr: "a.b",
		},
		{
			name: "format overrides content type", format: "json",
			ctype: "text/plain", body: `[1]`,
			want: []map[string]any{{"value": float64(1)}},
		},
		{
			name: "ndjson", ctype: "application/x-ndjson",
			body: "{\"a\":1}\n\n{\"a\":2}\n",
			want: []map[string]any{{"a": float64(1)}, {"a": float64(2)}},
		},
		{
			name: "ndjson bad line", ctype: "application/x-ndjson",
			body: "{\"a\":1}\nzzz\n", wantErr: "line 2",
		},
		{
			name: "csv bom and delimiter", ctype: "text/csv; charset=utf-8",
			delim: ";", body: "\xEF\xBB\xBFdate;count\n2026-10-01;5\n",
			want: []map[string]any{{"date": "2026-10-01", "count": "5"}},
		},
		{name: "csv header only", ctype: "application/csv", body: "a,b\n"},
		{
			name: "csv field mismatch", ctype: "text/csv",
			body: "a,b\n1,2\n3\n", wantErr: "line 3",
		},
		{
			name: "yaml list", ctype: "application/x-yaml", root: "items",
			body: "items:\n  - id: 1\n    tags: {env: prod}\n",
			want: []map[string]any{{"id": 1, "tags.env": "prod"}},
		},
		{
			name: "yaml non-string keys", ctype: "text/yaml",
			body: "1: {2: a}\n",
			want: []map[string]any{{"1.2": "a"}},
		},
		{
			name: "yaml missing root", ctype: "application/yaml",
			root: "a.b", body: "a: 1\n", wantErr: "a.b",
		},
		{
			name:  "xml rss",
			ctype: "application/xml",
			root:  "rss.channel.item",
			body:  `<rss><channel><item id="1"><title>A</title></item><item id="2"><title>B</title></item></channel></rss>`,
			want: []map[string]any{
				{"@id": "1", "title": "A"},
				{"@id": "2", "title": "B"},
			},
		},
		{
			name: "xml repeated children", ctype: "text/xml", root: "r.row",
			body: `<r><row><tag>a</tag><tag>b</tag></row></r>`,
			want: []map[string]any{{"tag.0": "a", "tag.1": "b"}},
		},
		{
			name: "xml no root, namespaces, text with attrs", ctype: "text/xml",
			body: `<x:a xmlns:x="urn:x"><x:b k="v"> t <x:c>1</x:c></x:b></x:a>`,
			want: []map[string]any{{"@k": "v", "#text": "t", "c": "1"}},
		},
		{
			name: "xml missing root", ctype: "application/xml",
			root: "rss.feed.item", body: `<rss><channel/></rss>`,
			wantErr: "rss.feed.item",
		},
		{
			name: "xml entity not expanded", ctype: "application/xml",
			body:    `<!DOCTYPE a [<!ENTITY e "boom">]><a><b>&e;</b></a>`,
			wantErr: "decode xml",
		},
		{
			name: "xlsx named sheet", ctype: xlsxType, sheet: "Data",
			body: xlsx,
			want: []map[string]any{
				{"date": "2026-10-01", "count": "5"},
				{"date": "2026-10-02", "count": ""},
			},
		},
		{
			name: "xlsx first sheet", format: "xlsx", body: xlsx,
			wantErr: "",
		},
		{
			name: "xlsx missing sheet", ctype: xlsxType, sheet: "Nope",
			body: xlsx, wantErr: "Nope",
		},
		{
			name: "text", ctype: "text/plain", body: "a\r\n\nb\n",
			want: []map[string]any{{"line": "a"}, {"line": "b"}},
		},
		{
			name: "unknown content type", ctype: "text/html",
			body: "<html>", wantErr: "text/html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := decode(
				tt.format,
				tt.ctype,
				tt.root,
				tt.delim,
				tt.sheet,
				[]byte(tt.body),
			)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestXMLDepth(t *testing.T) {
	t.Parallel()

	_, err := decode(
		"xml",
		"",
		"",
		"",
		"",
		[]byte(strings.Repeat("<a>", 100000)),
	)
	require.ErrorContains(t, err, "1000")

	ok := strings.Repeat(
		"<a>",
		maxXMLDepth,
	) + strings.Repeat(
		"</a>",
		maxXMLDepth,
	)

	_, err = decode("xml", "", "", "", "", []byte(ok))
	require.NoError(t, err)
}

// Не параллельный: меняет maxBody.
func TestXLSXSizeLimit(t *testing.T) {
	body := []byte(testXLSX(t))

	_, err := decode("xlsx", "", "", "", "", body)
	require.NoError(t, err)

	old := maxBody

	maxBody = 100
	t.Cleanup(func() { maxBody = old })

	_, err = decode("xlsx", "", "", "", "", body)
	require.Error(t, err)
}
