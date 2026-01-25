package text

import (
	"encoding/json"
	"fmt"
	htmltmpl "html/template"
	"strings"
	"text/template"
	"time"
)

var ruMonths = map[time.Month]string{
	time.January:   "января",
	time.February:  "февраля",
	time.March:     "марта",
	time.April:     "апреля",
	time.May:       "мая",
	time.June:      "июня",
	time.July:      "июля",
	time.August:    "августа",
	time.September: "сентября",
	time.October:   "октября",
	time.November:  "ноября",
	time.December:  "декабря",
}

var ruMonthsNominative = map[time.Month]string{
	time.January:   "январь",
	time.February:  "февраль",
	time.March:     "март",
	time.April:     "апрель",
	time.May:       "май",
	time.June:      "июнь",
	time.July:      "июль",
	time.August:    "август",
	time.September: "сентябрь",
	time.October:   "октябрь",
	time.November:  "ноябрь",
	time.December:  "декабрь",
}

var FuncMap = template.FuncMap{
	// 	upper - Преобразует строку в верхний регистр
	// Вход: "hello world"
	// Шаблон: {{ upper .text }}
	// Результат: "HELLO WORLD"
	"upper": strings.ToUpper,

	// 	lower - Преобразует строку в нижний регистр
	// Вход: "HELLO WORLD"
	// Шаблон: {{ lower .text }}
	// Результат: "hello world"
	"lower": strings.ToLower,

	// trim - Удаляет пробелы в начале и конце строки
	// Вход: "  hello  "
	// Шаблон: {{ trim .text }}
	// Результат: "hello"
	"trim": func(s string) string {
		return strings.TrimSpace(s)
	},

	// contains - Проверяет, содержит ли строка подстроку
	// Вход: s = "BANK Bank", substr = "BANK"
	// Шаблон: {{ if contains .bank "BANK" }}Это BANK{{ end }}
	// Результат: "Это BANK"
	"contains": func(s, substr string) bool {
		return strings.Contains(s, substr)
	},

	// hasPrefix - Проверяет, начинается ли строка с префикса
	// Вход: s = "Итого по банку", prefix = "Итого"
	// Шаблон: {{ if hasPrefix .text "Итого" }}Это итого{{ end }}
	// Результат: "Это итого"
	"hasPrefix": func(s, prefix string) bool {
		return strings.HasPrefix(s, prefix)
	},

	// hasSuffix - Проверяет, заканчивается ли строка суффиксом
	// Вход: s = "report.pdf", suffix = ".pdf"
	// Шаблон: {{ if hasSuffix .filename ".pdf" }}PDF файл{{ end }}
	// Результат: "PDF файл"
	"hasSuffix": func(s, suffix string) bool {
		return strings.HasSuffix(s, suffix)
	},

	// split - Разделяет строку по разделителю
	// Вход: s = "one,two,three", sep = ","
	// Шаблон: {{ range split .csv "," }}{{ . }} {{ end }}
	// Результат: "one two three "
	"split": func(s, sep string) []string {
		return strings.Split(s, sep)
	},

	// replace - Заменяет все вхождения подстроки
	// Вход: s = "hello world", old = "world", new = "Go"
	// Шаблон: {{ replace .text "world" "Go" }}
	// Результат: "hello Go"
	"replace": func(s, old, new string) string {
		return strings.ReplaceAll(s, old, new)
	},

	// money - Форматирует число как денежную сумму (простой формат)
	// Вход: 1234567.89
	// Шаблон: {{ money .amount }}
	// Результат: "1234567.89"
	"money": func(val any) string {
		var f float64
		switch v := val.(type) {
		case float64:
			f = v
		case float32:
			f = float64(v)
		case int:
			f = float64(v)
		case int64:
			f = float64(v)
		default:
			return "0.00"
		}
		return fmt.Sprintf("%.2f", f)
	},

	// formatMoney - Форматирует число как денежную сумму с разделителями тысяч
	// Вход: 1234567.89
	// Шаблон: {{ formatMoney .amount }}
	// Результат: "1 234 567.89"
	"formatMoney": func(val any) string {
		var f float64
		switch v := val.(type) {
		case float64:
			f = v
		case float32:
			f = float64(v)
		case int:
			f = float64(v)
		case int64:
			f = float64(v)
		default:
			return "0.00"
		}

		s := fmt.Sprintf("%.2f", f)
		parts := strings.Split(s, ".")
		intPart := parts[0]
		decPart := parts[1]

		var result strings.Builder
		for i, c := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result.WriteString(" ")
			}
			result.WriteRune(c)
		}

		return result.String() + "." + decPart
	},

	// formatNumber - Форматирует целое число с разделителями тысяч
	// Вход: 1234567
	// Шаблон: {{ formatNumber .count }}
	// Результат: "1 234 567"
	"formatNumber": func(val any) string {
		var n int64
		switch v := val.(type) {
		case int:
			n = int64(v)
		case int64:
			n = v
		case int32:
			n = int64(v)
		case float64:
			n = int64(v)
		case float32:
			n = int64(v)
		default:
			return "0"
		}

		s := fmt.Sprintf("%d", n)
		var result strings.Builder
		for i, c := range s {
			if i > 0 && (len(s)-i)%3 == 0 {
				result.WriteString(" ")
			}
			result.WriteRune(c)
		}

		return result.String()
	},

	// percent - Вычисляет процент от общего числа
	// Вход: part = 25, total = 100
	// Шаблон: {{ percent .part .total }}%
	// Результат: "25.00%"
	"percent": func(part, total any) float64 {
		var p, t float64

		switch v := part.(type) {
		case float64:
			p = v
		case int, int64:
			p = float64(v.(int))
		}

		switch v := total.(type) {
		case float64:
			t = v
		case int, int64:
			t = float64(v.(int))
		}

		if t == 0 {
			return 0
		}
		return (p / t) * 100
	},

	// yesterday - Возвращает вчерашнюю дату
	// Вход: 2026-01-25
	// Шаблон: {{ yesterday .date }}
	// Результат: 2026-01-24
	"yesterday": func(t time.Time) time.Time {
		return t.AddDate(0, 0, -1)
	},

	// formatDateDMY - Возращает строку с форматированной датой
	// Вход: 2026-01-25
	// Шаблон: {{ .date | formatDateDMY }}
	// Результат: 25.01.2026
	"formatDateDMY": func(t time.Time) string {
		return t.Format("02.01.2006")
	},

	// lastMonth - Возвращает дату месяц назад
	// Вход: 2026-01-25
	// Шаблон: {{ lastMonth .date }}
	// Результат: 2025-12-25
	"lastMonth": func(t time.Time) time.Time {
		return t.AddDate(0, -1, 0)
	},

	// lastYear - Возвращает дату год назад
	// Вход: 2026-01-25
	// Шаблон: {{ lastYear .date }}
	// Результат: 2025-01-25
	"lastYear": func(t time.Time) time.Time {
		return t.AddDate(-1, 0, 0)
	},

	// addDays - Добавляет дни к дате
	// Вход: date = 2026-01-25, days = 5
	// Шаблон: {{ addDays .date 5 }}
	// Результат: 2026-01-30
	"addDays": func(t time.Time, days int) time.Time {
		return t.AddDate(0, 0, days)
	},

	// subDays - Вычитает дни из даты
	// Вход: date = 2026-01-25, days = 5
	// Шаблон: {{ subDays .date 5 }}
	// Результат: 2026-01-20
	"subDays": func(t time.Time, days int) time.Time {
		return t.AddDate(0, 0, -days)
	},

	// diffDays - Возвращает разницу в днях между датами
	// Вход: a = 2026-01-30, b = 2026-01-25
	// Шаблон: {{ diffDays .dateA .dateB }} дней
	// Результат: "5 дней"
	"diffDays": func(a, b time.Time) int {
		return int(a.Sub(b).Hours() / 24)
	},

	// addDuration - Добавляет временной интервал к дате
	// Вход: date = 2026-01-25 10:00, duration = "2h30m"
	// Шаблон: {{ addDuration .date "2h30m" }}
	// Результат: 2026-01-25 12:30
	"addDuration": func(t time.Time, d string) (time.Time, error) {
		dd, err := time.ParseDuration(d)
		if err != nil {
			return time.Time{}, err
		}

		return t.Add(dd), nil
	},

	// startOfDay - Возвращает начало дня (00:00:00)
	// Вход: 2026-01-25 15:30:45
	// Шаблон: {{ startOfDay .date }}
	// Результат: 2026-01-25 00:00:00
	"startOfDay": func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	},

	// startOfMonth - Возвращает первый день месяца
	// Вход: 2026-01-25
	// Шаблон: {{ startOfMonth .date }}
	// Результат: 2026-01-01
	"startOfMonth": func(t time.Time) time.Time {
		y, m, _ := t.Date()
		return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
	},

	// endOfMonth - Возвращает последний момент месяца
	// Вход: 2026-01-25
	// Шаблон: {{ endOfMonth .date }}
	// Результат: 2026-01-31 23:59:59
	"endOfMonth": func(t time.Time) time.Time {
		y, m, _ := t.Date()
		return time.Date(y, m+1, 1, 0, 0, 0, 0, t.Location()).Add(-time.Nanosecond)
	},

	// startOfYear - Возвращает первый день года
	// Вход: 2026-01-25
	// Шаблон: {{ startOfYear .date }}
	// Результат: 2026-01-01
	"startOfYear": func(t time.Time) time.Time {
		y := t.Year()
		return time.Date(y, 1, 1, 0, 0, 0, 0, t.Location())
	},

	// endOfYear - Возвращает последний момент года
	// Вход: 2026-01-25
	// Шаблон: {{ endOfYear .date }}
	// Результат: 2026-12-31 23:59:59
	"endOfYear": func(t time.Time) time.Time {
		return time.Date(t.Year()+1, 1, 1, 0, 0, 0, 0, t.Location()).
			Add(-time.Nanosecond)
	},

	// formatRuDate - Форматирует дату на русском языке
	// Вход: 2026-01-25
	// Шаблон: {{ formatRuDate .date }}
	// Результат: "25 января 2026"
	"formatRuDate": func(t time.Time) string {
		day := t.Day()
		month := ruMonths[t.Month()]
		year := t.Year()

		return fmt.Sprintf("%02d %s %d", day, month, year)
	},

	// formatRuMonthYear - Форматирует месяц и год на русском языке
	// Вход: 2026-01-25
	// Шаблон: {{ formatRuMonthYear .date }}
	// Результат: "январь 2026"
	"formatRuMonthYear": func(t time.Time) string {
		month := ruMonthsNominative[t.Month()]
		year := t.Year()

		return fmt.Sprintf("%s %d", month, year)
	},

	// formatRuDateTime - Форматирует дату и время на русском языке
	// Вход: 2026-01-25 15:30
	// Шаблон: {{ formatRuDateTime .date }}
	// Результат: "25 января 2026, 15:30"
	"formatRuDateTime": func(t time.Time) string {
		day := t.Day()
		month := ruMonths[t.Month()]
		year := t.Year()
		hour, min := t.Hour(), t.Minute()

		return fmt.Sprintf("%02d %s %d, %02d:%02d", day, month, year, hour, min)
	},

	// formatDateShort - Форматирует дату в кратком формате
	// Вход: 2026-01-25
	// Шаблон: {{ formatDateShort .date }}
	// Результат: "25.01.2026"
	"formatDateShort": func(t time.Time) string {
		return t.Format("02.01.2006")
	},

	// formatDateTime - Форматирует дату и время в кратком формате
	// Вход: 2026-01-25 15:30
	// Шаблон: {{ formatDateTime .date }}
	// Результат: "25.01.2026 15:30"
	"formatDateTime": func(t time.Time) string {
		return t.Format("02.01.2006 15:04")
	},

	// monthRu - Возвращает название месяца на русском (именительный падеж)
	// Вход: 2026-01-25
	// Шаблон: {{ monthRu .date }}
	// Результат: "январь"
	"monthRu": func(t time.Time) string {
		return ruMonthsNominative[t.Month()]
	},

	// monthRuGenitive - Возвращает название месяца на русском (родительный падеж)
	// Вход: 2026-01-25
	// Шаблон: {{ monthRuGenitive .date }}
	// Результат: "января"
	"monthRuGenitive": func(t time.Time) string {
		return ruMonths[t.Month()]
	},

	// monthNameRu - Возвращает название месяца по номеру (именительный падеж)
	// Вход: 1
	// Шаблон: {{ monthNameRu 1 }}
	// Результат: "январь"
	"monthNameRu": func(monthNum int) string {
		months := map[int]string{
			1:  "январь",
			2:  "февраль",
			3:  "март",
			4:  "апрель",
			5:  "май",
			6:  "июнь",
			7:  "июль",
			8:  "август",
			9:  "сентябрь",
			10: "октябрь",
			11: "ноябрь",
			12: "декабрь",
		}
		if name, ok := months[monthNum]; ok {
			return name
		}
		return ""
	},

	// monthNameRuGenitive - Возвращает название месяца по номеру (родительный падеж)
	// Вход: 1
	// Шаблон: {{ monthNameRuGenitive 1 }}
	// Результат: "января"
	"monthNameRuGenitive": func(monthNum int) string {
		months := map[int]string{
			1:  "января",
			2:  "февраля",
			3:  "марта",
			4:  "апреля",
			5:  "мая",
			6:  "июня",
			7:  "июля",
			8:  "августа",
			9:  "сентября",
			10: "октября",
			11: "ноября",
			12: "декабря",
		}
		if name, ok := months[monthNum]; ok {
			return name
		}
		return ""
	},

	// get - Получает значение из map по ключу
	// Вход: map = {"key": "value"}
	// Шаблон: {{ get . "key" }}
	// Результат: "value"
	"get": func(m map[string]any, key string) any {
		if m == nil {
			return nil
		}
		return m[key]
	},

	// getOr - Получает значение из map или возвращает значение по умолчанию
	// Вход: map = {"key": "value"}
	// Шаблон: {{ getOr . "missing" "default" }}
	// Результат: "default"
	"getOr": func(m map[string]any, key string, def any) any {
		if m == nil {
			return def
		}
		val, ok := m[key]
		if !ok || val == nil {
			return def
		}
		return val
	},

	// hasKey - Проверяет наличие ключа в map
	// Вход: map = {"key": "value"}
	// Шаблон: {{ if hasKey . "key" }}Ключ есть{{ end }}
	// Результат: "Ключ есть"
	"hasKey": func(m map[string]any, key string) bool {
		if m == nil {
			return false
		}
		_, ok := m[key]
		return ok
	},

	// keys - Возвращает все ключи из map
	// Вход: map = {"a": 1, "b": 2}
	// Шаблон: {{ range keys . }}{{ . }} {{ end }}
	// Результат: "a b "
	"keys": func(m map[string]any) []string {
		if m == nil {
			return []string{}
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		return keys
	},

	// values - Возвращает все значения из map
	// Вход: map = {"a": 1, "b": 2}
	// Шаблон: {{ range values . }}{{ . }} {{ end }}
	// Результат: "1 2 "
	"values": func(m map[string]any) []any {
		if m == nil {
			return []any{}
		}
		vals := make([]any, 0, len(m))
		for _, v := range m {
			vals = append(vals, v)
		}
		return vals
	},

	// first - Возвращает первый элемент массива
	// Вход: [1, 2, 3]
	// Шаблон: {{ first .array }}
	// Результат: 1
	"first": func(arr any) any {
		switch v := arr.(type) {
		case []any:
			if len(v) > 0 {
				return v[0]
			}
		case []map[string]any:
			if len(v) > 0 {
				return v[0]
			}
		}
		return nil
	},

	// last - Возвращает последний элемент массива
	// Вход: [1, 2, 3]
	// Шаблон: {{ last .array }}
	// Результат: 3
	"last": func(arr any) any {
		switch v := arr.(type) {
		case []any:
			if len(v) > 0 {
				return v[len(v)-1]
			}
		case []map[string]any:
			if len(v) > 0 {
				return v[len(v)-1]
			}
		}
		return nil
	},

	// size - Возвращает размер массива, map или строки
	// Вход: [1, 2, 3]
	// Шаблон: {{ size .array }}
	// Результат: 3
	"size": func(val any) int {
		switch v := val.(type) {
		case []any:
			return len(v)
		case []map[string]any:
			return len(v)
		case map[string]any:
			return len(v)
		case string:
			return len(v)
		default:
			return 0
		}
	},

	// isEmpty - Проверяет, пустой ли массив, map или строка
	// Вход: []
	// Шаблон: {{ if isEmpty .array }}Пусто{{ end }}
	// Результат: "Пусто"
	"isEmpty": func(val any) bool {
		switch v := val.(type) {
		case []any:
			return len(v) == 0
		case []map[string]any:
			return len(v) == 0
		case map[string]any:
			return len(v) == 0
		case string:
			return v == "" || v == "null"
		case nil:
			return true
		default:
			return false
		}
	},

	// toJson - Преобразует значение в JSON строку
	// Вход: {"name": "John", "age": 30}
	// Шаблон: {{ toJson . }}
	// Результат: '{"name":"John","age":30}'
	"toJson": func(v any) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},

	// parseJson - Парсит JSON строку в значение
	// Вход: '{"name": "John"}'
	// Шаблон: {{ $data := parseJson .json }}{{ $data.name }}
	// Результат: "John"
	"parseJson": func(s string) (any, error) {
		var result any
		err := json.Unmarshal([]byte(s), &result)
		return result, err
	},

	// parseJsonMap - Парсит JSON строку в map
	// Вход: '{"promo1": 5, "promo2": 10}'
	// Шаблон: {{ $promos := parseJsonMap .json }}{{ range $k, $v := $promos }}{{ $k }}: {{ $v }} {{ end }}
	// Результат: "promo1: 5 promo2: 10 "
	"parseJsonMap": func(s string) (map[string]any, error) {
		var result map[string]any
		if s == "" || s == "null" {
			return nil, nil
		}
		err := json.Unmarshal([]byte(s), &result)
		return result, err
	},

	// ternary - Тернарный оператор (условие ? истина : ложь)
	// Вход: condition = true, trueVal = "Да", falseVal = "Нет"
	// Шаблон: {{ ternary .condition "Да" "Нет" }}
	// Результат: "Да"
	"ternary": func(condition bool, trueVal, falseVal any) any {
		if condition {
			return trueVal
		}
		return falseVal
	},

	// coalesce - Возвращает первое не-пустое значение
	// Вход: values = ["", null, "first", "second"]
	// Шаблон: {{ coalesce "" nil "first" "second" }}
	// Результат: "first"
	"coalesce": func(values ...any) any {
		for _, v := range values {
			if v != nil && v != "" && v != "null" {
				return v
			}
		}
		return nil
	},

	// safe - Помечает строку как безопасный HTML (не экранируется)
	// Вход: "<b>Bold</b>"
	// Шаблон: {{ safe .html }}
	// Результат: <b>Bold</b> (отрендерится как HTML)
	"safe": func(s string) htmltmpl.HTML {
		return htmltmpl.HTML(s)
	},
	// isZero - Проверяет, является ли значение нулевым
	// Вход: 0
	// Шаблон: {{ if isZero .count }}Ноль{{ end }}
	// Результат: "Ноль"
	"isZero": func(val any) bool {
		switch v := val.(type) {
		case int, int64, int32, float64, float32:
			return v == 0
		case string:
			return v == "" || v == "0"
		case nil:
			return true
		default:
			return false
		}
	},

	// notEmpty - Проверяет, что строка не пустая и не "null"
	// Вход: "hello"
	// Шаблон: {{ if notEmpty .text }}Есть текст{{ end }}
	// Результат: "Есть текст"
	"notEmpty": func(s string) bool {
		return s != "" && s != "null"
	},

	// default - Возвращает значение по умолчанию, если значение пустое
	// Вход: ""
	// Шаблон: {{ default "No data" .text }}
	// Результат: "No data"
	"default": func(def, val any) any {
		if val == nil || val == "" || val == "null" {
			return def
		}
		return val
	},

	// escape - Экранирует специальные символы Telegram MarkdownV2
	// Вход: "Hello_World*Test"
	// Шаблон: {{ escape .text }}
	// Результат: "Hello\_World\*Test"
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

	// escapeHTML - Экранирует HTML специальные символы
	// Вход: "<script>alert('XSS')</script>"
	// Шаблон: {{ escapeHTML .html }}
	// Результат: "&lt;script&gt;alert('XSS')&lt;/script&gt;"
	"escapeHTML": func(s string) string {
		r := strings.NewReplacer(
			"&", "&amp;",
			"<", "&lt;",
			">", "&gt;",
			`"`, "&quot;",
		)
		return r.Replace(s)
	},

	// bold - Оборачивает текст в HTML тег <b>
	// Вход: "важный текст"
	// Шаблон: {{ bold "важный текст" }}
	// Результат: "<b>важный текст</b>"
	"bold": func(s string) string {
		return "<b>" + s + "</b>"
	},

	// italic - Оборачивает текст в HTML тег <i>
	// Вход: "курсив"
	// Шаблон: {{ italic "курсив" }}
	// Результат: "<i>курсив</i>"
	"italic": func(s string) string {
		return "<i>" + s + "</i>"
	},

	// code - Оборачивает текст в HTML тег <code>
	// Вход: "console.log()"
	// Шаблон: {{ code "console.log()" }}
	// Результат: "<code>console.log()</code>"
	"code": func(s string) string {
		return "<code>" + s + "</code>"
	},
}
