package text

import (
	"strings"
	"text/template"
	"time"
)

var FuncMap = template.FuncMap{
	"upper": strings.ToUpper,
	"lower": strings.ToLower,

	"addDays": func(t time.Time, days int) time.Time {
		return t.AddDate(0, 0, days)
	},
	"subDays": func(t time.Time, days int) time.Time {
		return t.AddDate(0, 0, -days)
	},
	"diffDays": func(a, b time.Time) int {
		return int(a.Sub(b).Hours() / 24)
	},
	"addDuration": func(t time.Time, d string) (time.Time, error) {
		dd, err := time.ParseDuration(d)
		if err != nil {
			return time.Time{}, err
		}

		return t.Add(dd), nil
	},
	"escape": func(s string) string {
		r := strings.NewReplacer(
			"_", `\_`,
			"*", `\*`,
			"[", `\[`,
			"]", `\]`,
			"(", `\(`,
			")", `\)`,
			"~", `\~`,
			"`", "\\`",
			">", `\>`,
			"#", `\#`,
			"+", `\+`,
			"-", `\-`,
			"=", `\=`,
			"|", `\|`,
			"{", `\{`,
			"}", `\}`,
			".", `\.`,
			"!", `\!`,
		)

		return r.Replace(s)
	},
}
