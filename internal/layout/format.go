// Package layout is the pure core of the export pipeline: it validates and
// normalizes a models.Layout, resolves each block against a dataset, packs the
// blocks onto pages, and formats individual values. It imports only
// internal/models and the standard library so the CLI validator, the live
// preview server and unit tests never pull in a rendering engine.
package layout

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FormatValue renders a single cell or metric value for every output format. It
// is pure and shared by the html, csv, xlsx, png and text renderers so a column
// or metric "format" behaves identically everywhere.
//
//	""        nil -> ""; whole numbers without a fraction; other floats to at
//	          most two decimals; anything else via fmt.Sprint
//	"number"  integer with space-grouped thousands
//	"percent" value*100 with at most one decimal and a "%" suffix
//	"money"   two decimals with a space-grouped integer part
//
// An unknown format is treated as "".
func FormatValue(v any, format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "number":
		if f, ok := toFloat(v); ok {
			return groupThousands(strconv.FormatInt(int64(math.Round(f)), 10))
		}

		return fallbackString(v)
	case "percent":
		if f, ok := toFloat(v); ok {
			return trimFractionZeros(strconv.FormatFloat(f*100, 'f', 1, 64)) + "%"
		}

		return fallbackString(v)
	case "money":
		if f, ok := toFloat(v); ok {
			parts := strings.SplitN(strconv.FormatFloat(f, 'f', 2, 64), ".", 2)

			return groupThousands(parts[0]) + "." + parts[1]
		}

		return fallbackString(v)
	default:
		return fallbackString(v)
	}
}

func fallbackString(v any) string {
	switch n := v.(type) {
	case nil:
		return ""
	case string:
		return n
	case float64:
		return trimFloat(n)
	case float32:
		return trimFloat(float64(n))
	default:
		return fmt.Sprint(v)
	}
}

// trimFloat prints an integral float without a fraction and any other float with
// at most two decimals, trailing zeros removed.
func trimFloat(f float64) string {
	if !math.IsInf(f, 0) && !math.IsNaN(f) && f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}

	return trimFractionZeros(strconv.FormatFloat(f, 'f', 2, 64))
}

// trimFractionZeros drops a trailing fractional zero run: "89.0" -> "89",
// "89.50" -> "89.5", "89" -> "89".
func trimFractionZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}

	s = strings.TrimRight(s, "0")

	return strings.TrimSuffix(s, ".")
}

// groupThousands inserts a space every three digits from the right, mirroring
// internal/pkg/text.formatNumber.
func groupThousands(intText string) string {
	sign := ""
	if strings.HasPrefix(intText, "-") {
		sign, intText = "-", intText[1:]
	}

	var b strings.Builder

	for i, r := range intText {
		if i > 0 && (len(intText)-i)%3 == 0 {
			b.WriteByte(' ')
		}

		b.WriteRune(r)
	}

	return sign + b.String()
}
