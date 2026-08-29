package text

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

var textMap = map[string]any{
	"rich_table": richTable,
}

func richTable(caption string, data []map[string]any, titleRow []string) template.HTML {
	title, labels := parseTitleRow(titleRow)
	var buf bytes.Buffer

	buf.WriteString("<table>\n")
	if caption != "" {
		buf.WriteString("<caption>")
		buf.WriteString(caption)
		buf.WriteString("</caption>\n")
	}

	buf.WriteString("<tr>\n")
	for _, t := range title {
		buf.WriteString("    <td>")
		buf.WriteString(t)
		buf.WriteString("</td>\n")
	}
	buf.WriteString("</tr>\n")

	for _, row := range data {
		buf.WriteString("<tr>\n")
		for _, label := range labels {
			buf.WriteString("    <td>")
			fmt.Fprint(&buf, row[label])
			buf.WriteString("</td>\n")
		}
		buf.WriteString("</tr>\n")
	}

	buf.WriteString("</table>\n")
	return template.HTML(buf.String())
}

func parseTitleRow(tRow []string) (title []string, labels []string) {
	for _, t := range tRow {
		parts := strings.Split(t, ":")
		if len(parts) == 2 {
			labels = append(labels, parts[0])
			title = append(title, parts[1])
		}
	}
	return
}
