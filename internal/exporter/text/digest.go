package text

import (
	"bytes"
	"fmt"
	"strings"
	texttemplate "text/template"
	"unicode/utf8"

	"support_bot/internal/models"

	layoutpkg "support_bot/internal/layout"
)

// renderDigest builds a plain-text summary of a layout that carries no text
// block, so the text format still produces something useful for any report:
// the page header, metrics as "label: value", tables as aligned columns and
// charts as "title: series=last-value".
func renderDigest(
	doc layoutpkg.ResolvedDocument,
	data models.Dataset,
	fm map[string]any,
) (string, error) {
	var sections []string

	if header := headerLine(doc.Page.Header, data, fm); header != "" {
		sections = append(sections, header)
	}

	var metrics []string

	flushMetrics := func() {
		if len(metrics) > 0 {
			sections = append(sections, strings.Join(metrics, "   "))
			metrics = nil
		}
	}

	for _, rb := range doc.Blocks {
		switch rb.Block.Type {
		case models.BlockTypeMetric:
			if line := metricLine(rb); line != "" {
				metrics = append(metrics, line)
			}
		case models.BlockTypeTable:
			flushMetrics()

			if table := tableSection(rb); table != "" {
				sections = append(sections, table)
			}
		case models.BlockTypeChart:
			flushMetrics()

			if line := chartLine(rb); line != "" {
				sections = append(sections, line)
			}
		default:
			// text blocks never reach the digest (their presence disables it);
			// raw_html is markup and is skipped.
		}
	}

	flushMetrics()

	if len(sections) == 0 {
		return "", fmt.Errorf("text: layout has nothing to summarise")
	}

	return strings.Join(sections, "\n\n") + "\n", nil
}

func headerLine(sec *models.PageSection, data models.Dataset, fm map[string]any) string {
	if sec == nil || strings.TrimSpace(sec.Text) == "" {
		return ""
	}

	t, err := texttemplate.New("digest_header").Funcs(fm).Parse(sec.Text)
	if err != nil {
		return strings.TrimSpace(sec.Text)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return strings.TrimSpace(sec.Text)
	}

	return strings.TrimSpace(buf.String())
}

func metricLine(rb layoutpkg.ResolvedBlock) string {
	mb := rb.Block.Metric
	if mb == nil {
		return ""
	}

	label := mb.Label
	if label == "" {
		label = mb.Field
	}

	value := ""

	for _, row := range rb.Rows {
		if v, ok := row[mb.Field]; ok {
			value = layoutpkg.FormatValue(v, mb.Format)

			break
		}
	}

	return label + ": " + value
}

func chartLine(rb layoutpkg.ResolvedBlock) string {
	ch := rb.Block.Chart
	if ch == nil || len(rb.Rows) == 0 {
		return ""
	}

	last := rb.Rows[len(rb.Rows)-1]

	parts := make([]string, 0, len(ch.Series))
	for _, s := range ch.Series {
		parts = append(parts, s.TitleOrField()+"="+layoutpkg.FormatValue(last[s.Field], ""))
	}

	title := strings.TrimSpace(ch.Title)
	if title == "" {
		title = string(ch.Kind) + " chart"
	}

	return title + ": " + strings.Join(parts, " ")
}

func tableSection(rb layoutpkg.ResolvedBlock) string {
	tb := rb.Block.Table
	if tb == nil || len(tb.Columns) == 0 {
		return ""
	}

	headers := make([]string, len(tb.Columns))
	widths := make([]int, len(tb.Columns))

	for i, c := range tb.Columns {
		headers[i] = c.Title
		if headers[i] == "" {
			headers[i] = c.Field
		}

		widths[i] = utf8.RuneCountInString(headers[i])
	}

	cells := make([][]string, len(rb.Rows))

	for r, row := range rb.Rows {
		cells[r] = make([]string, len(tb.Columns))

		for i, c := range tb.Columns {
			v := layoutpkg.FormatValue(row[c.Field], c.Format)
			cells[r][i] = v

			if w := utf8.RuneCountInString(v); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder

	writeRow := func(vals []string) {
		for i, v := range vals {
			if i > 0 {
				b.WriteString("  ")
			}

			b.WriteString(v)

			if i < len(vals)-1 {
				b.WriteString(strings.Repeat(" ", widths[i]-utf8.RuneCountInString(v)))
			}
		}

		b.WriteByte('\n')
	}

	writeRow(headers)

	for _, row := range cells {
		writeRow(row)
	}

	return strings.TrimRight(b.String(), "\n")
}
