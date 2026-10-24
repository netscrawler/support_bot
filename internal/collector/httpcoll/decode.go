package httpcoll

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"support_bot/internal/pkg/funcs"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

const (
	formatJSON   = "json"
	formatNDJSON = "ndjson"
	formatCSV    = "csv"
	formatXML    = "xml"
	formatXLSX   = "xlsx"
	formatYAML   = "yaml"
	formatText   = "text"

	// maxXMLDepth ограничивает вложенность: иначе тело в пределах maxBody раздувает дерево узлов.
	maxXMLDepth = 1000
)

// decode превращает тело ответа в строки; format "" — по Content-Type.
func decode(
	format, contentType, root, delimiter, sheet string,
	body []byte,
) ([]map[string]any, error) {
	if format == "" {
		mt, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, fmt.Errorf("unsupported content type %q", contentType)
		}

		switch mt {
		case "application/json":
			format = formatJSON
		case "application/x-ndjson":
			format = formatNDJSON
		case "text/csv", "application/csv":
			format = formatCSV
		case "application/xml", "text/xml":
			format = formatXML
		case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
			format = formatXLSX
		case "application/yaml", "application/x-yaml", "text/yaml":
			format = formatYAML
		case "text/plain":
			format = formatText
		default:
			return nil, fmt.Errorf("unsupported content type %q", contentType)
		}
	}

	switch format {
	case formatJSON:
		return decodeJSON(body, root)
	case formatNDJSON:
		return decodeNDJSON(body)
	case formatCSV:
		return decodeCSV(body, delimiter)
	case formatXML:
		return decodeXML(body, root)
	case formatXLSX:
		return decodeXLSX(body, sheet)
	case formatYAML:
		return decodeYAML(body, root)
	case formatText:
		return decodeText(body)
	}

	return nil, fmt.Errorf("unknown format %q", format)
}

// row приводит значение к строке результата: объект разворачивается, скаляр уходит в value.
func row(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return funcs.Flatten(m)
	}

	return map[string]any{"value": v}
}

func decodeJSON(body []byte, root string) ([]map[string]any, error) {
	var doc any

	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	return docRows(doc, root, "json")
}

// docRows идёт по root (ключи через точку) и превращает найденное значение в строки.
func docRows(doc any, root, format string) ([]map[string]any, error) {
	if root != "" {
		for key := range strings.SplitSeq(root, ".") {
			m, ok := doc.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s root %q not found", format, root)
			}

			if doc, ok = m[key]; !ok {
				return nil, fmt.Errorf("%s root %q not found", format, root)
			}
		}
	}

	items, ok := doc.([]any)
	if !ok {
		return []map[string]any{row(doc)}, nil
	}

	rows := make([]map[string]any, 0, len(items))
	for _, it := range items {
		rows = append(rows, row(it))
	}

	return rows, nil
}

func decodeNDJSON(body []byte) ([]map[string]any, error) {
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), len(body)+1)

	var rows []map[string]any

	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}

		var m map[string]any

		if err := json.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("ndjson line %d: %w", n, err)
		}

		rows = append(rows, funcs.Flatten(m))
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read ndjson: %w", err)
	}

	return rows, nil
}

func decodeCSV(body []byte, delimiter string) ([]map[string]any, error) {
	r := csv.NewReader(
		bytes.NewReader(bytes.TrimPrefix(body, []byte("\xEF\xBB\xBF"))),
	)

	if delimiter != "" {
		r.Comma, _ = utf8.DecodeRuneInString(delimiter)
	}

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("decode csv: %w", err)
	}

	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	rows := make([]map[string]any, 0, len(records)-1)

	for _, rec := range records[1:] {
		m := make(map[string]any, len(header))
		for i, h := range header {
			m[h] = rec[i]
		}

		rows = append(rows, m)
	}

	return rows, nil
}

func decodeYAML(body []byte, root string) ([]map[string]any, error) {
	var doc any

	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	return docRows(normalizeYAML(doc), root, "yaml")
}

// normalizeYAML приводит map[any]any (ключи не-строки) к map[string]any, как у json.
func normalizeYAML(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			x[k] = normalizeYAML(val)
		}

		return x

	case map[any]any:
		m := make(map[string]any, len(x))
		for k, val := range x {
			m[fmt.Sprint(k)] = normalizeYAML(val)
		}

		return m

	case []any:
		for i, val := range x {
			x[i] = normalizeYAML(val)
		}
	}

	return v
}

func decodeText(body []byte) ([]map[string]any, error) {
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), len(body)+1)

	var rows []map[string]any

	for sc.Scan() {
		if line := strings.TrimSuffix(sc.Text(), "\r"); line != "" {
			rows = append(rows, map[string]any{"line": line})
		}
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read text: %w", err)
	}

	return rows, nil
}

func decodeXLSX(body []byte, sheet string) ([]map[string]any, error) {
	f, err := excelize.OpenReader(bytes.NewReader(body), excelize.Options{
		UnzipSizeLimit:    maxBody,
		UnzipXMLSizeLimit: maxBody,
	})
	if err != nil {
		return nil, fmt.Errorf("decode xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if sheet == "" && len(sheets) > 0 {
		sheet = sheets[0]
	}

	if !slices.Contains(sheets, sheet) {
		return nil, fmt.Errorf("xlsx sheet %q not found", sheet)
	}

	records, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read xlsx sheet %q: %w", sheet, err)
	}

	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	rows := make([]map[string]any, 0, len(records)-1)

	for _, rec := range records[1:] {
		if !slices.ContainsFunc(rec, func(c string) bool { return c != "" }) {
			continue
		}

		m := make(map[string]any, len(header))
		for i, h := range header {
			m[h] = ""
			if i < len(rec) {
				m[h] = rec[i]
			}
		}

		rows = append(rows, m)
	}

	return rows, nil
}

// xnode — элемент XML: имя без префикса пространства имён, атрибуты, дети и текст.
type xnode struct {
	name  string
	attrs []xml.Attr
	kids  []*xnode
	text  strings.Builder
}

// value: лист без атрибутов — строка; иначе map (повторяющиеся дети — под индексами "0", "1", ...,
// чтобы Flatten дал tag.0, tag.1 даже для скаляров).
func (n *xnode) value() any {
	text := strings.TrimSpace(n.text.String())
	if len(n.kids) == 0 && len(n.attrs) == 0 {
		return text
	}

	m := make(map[string]any)

	for _, a := range n.attrs {
		m["@"+a.Name.Local] = a.Value
	}

	count := make(map[string]int)
	for _, k := range n.kids {
		count[k.name]++
	}

	seen := make(map[string]int)

	for _, k := range n.kids {
		if count[k.name] == 1 {
			m[k.name] = k.value()

			continue
		}

		g, ok := m[k.name].(map[string]any)
		if !ok {
			g = make(map[string]any)
			m[k.name] = g
		}

		g[strconv.Itoa(seen[k.name])] = k.value()
		seen[k.name]++
	}

	if text != "" {
		m["#text"] = text
	}

	return m
}

func parseXML(body []byte) (*xnode, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))

	var (
		stack []*xnode
		doc   *xnode
	)

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("decode xml: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if len(stack) >= maxXMLDepth {
				return nil, fmt.Errorf(
					"xml nesting deeper than %d",
					maxXMLDepth,
				)
			}

			n := &xnode{name: t.Name.Local}

			for _, a := range t.Attr {
				if a.Name.Space != "xmlns" && a.Name.Local != "xmlns" {
					n.attrs = append(n.attrs, a)
				}
			}

			if len(stack) > 0 {
				p := stack[len(stack)-1]

				p.kids = append(p.kids, n)
			} else if doc == nil {
				doc = n
			}

			stack = append(stack, n)

		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write(t)
			}
		}
	}

	if doc == nil {
		return nil, errors.New("decode xml: no root element")
	}

	return doc, nil
}

func decodeXML(body []byte, root string) ([]map[string]any, error) {
	doc, err := parseXML(body)
	if err != nil {
		return nil, err
	}

	// Без root строки — дети корневого элемента.
	level := doc.kids

	if root != "" {
		segs := strings.Split(root, ".")
		if segs[0] != doc.name {
			return nil, fmt.Errorf(
				"xml root %q not found (document root is %q)",
				root,
				doc.name,
			)
		}

		cur := []*xnode{doc}

		for _, seg := range segs[1:] {
			var next []*xnode

			for _, n := range cur {
				for _, k := range n.kids {
					if k.name == seg {
						next = append(next, k)
					}
				}
			}

			cur = next
		}

		if len(cur) == 0 {
			return nil, fmt.Errorf("xml root %q not found", root)
		}

		level = cur
	}

	rows := make([]map[string]any, 0, len(level))
	for _, n := range level {
		rows = append(rows, row(n.value()))
	}

	return rows, nil
}
