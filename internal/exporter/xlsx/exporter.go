package xlsx

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"support_bot/internal/models"
	"support_bot/internal/pkg"
	"time"

	"github.com/xuri/excelize/v2"
)

type Exporter struct{}

func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	var (
		buf *bytes.Buffer
		err error
	)

	if format.Layout != nil {
		buf, err = e.createXlsxBookFromLayout(data, *format.Layout)
	} else {
		buf, err = e.createXlsxBook(data, format.Order)
	}
	if err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(buf, *format.FileName+".xlsx")
	if err != nil {
		return nil, err
	}

	return []models.Data{fd}, nil
}

func (e *Exporter) createXlsxBook(
	dataMap map[string][]map[string]any,
	ordering map[string][]string,
) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	dateStyle, _ := f.NewStyle(&excelize.Style{CustomNumFmt: new("dd.mm.yyyy")})
	dateTimeStyle, _ := f.NewStyle(&excelize.Style{CustomNumFmt: new("dd.mm.yyyy hh:mm:ss")})

	styles := map[string]int{
		"date":     dateStyle,
		"datetime": dateTimeStyle,
	}

	for unit, records := range dataMap {
		if len(records) == 0 {
			continue
		}

		_, data := splitMeta(records)

		// Игнорируем все листы начинающиеся с _ чтобы не попадали в файл
		if strings.HasPrefix(unit, "_") {
			continue
		}

		var order []string
		if o, ok := ordering[unit]; ok {
			order = o
		} else {
			order = nil
		}

		sortedRecords := pkg.ConvertSortedRows(data, order)

		sheetName := sanitizeSheetName(unit)
		f.NewSheet(sheetName)

		for rowIdx, row := range sortedRecords {
			for colIdx, val := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				v, kind := detectValueType(fmt.Sprint(val))
				f.SetCellValue(sheetName, cell, v)
				if styleID, ok := styles[kind]; ok {
					f.SetCellStyle(sheetName, cell, cell, styleID)
				}
			}
		}

		// Добавление таблицы (с фильтрацией)
		startCell, _ := excelize.CoordinatesToCellName(1, 1)
		endCell, _ := excelize.CoordinatesToCellName(len(sortedRecords[0]), len(sortedRecords))
		tableRange := fmt.Sprintf("%s:%s", startCell, endCell)

		a := true

		err := f.AddTable(sheetName, &excelize.Table{
			Range:             tableRange,
			Name:              sheetName,
			StyleName:         "TableStyleMedium9",
			ShowColumnStripes: false,
			ShowFirstColumn:   false,
			ShowHeaderRow:     &a,
			ShowLastColumn:    false,
			ShowRowStripes:    &a,
		})
		if err != nil {
			return nil, fmt.Errorf("ошибка создания таблицы на листе %s: %w", sheetName, err)
		}

		// Автоширина колонок
		for colIdx := range sortedRecords[0] {
			colLetter, _ := excelize.ColumnNumberToName(colIdx + 1)
			colRange := colLetter + ":" + colLetter
			f.SetColWidth(sheetName, colRange, colRange, getAutoWidth(sortedRecords, colIdx))
		}
	}
	f.DeleteSheet("Sheet1")

	f.SetAppProps(&excelize.AppProperties{
		Application: "SendyStats",
		Company:     "Sendy",
	})

	return f.WriteToBuffer()
}

// detectValueType определяет тип значения по строке и возвращает подходящий тип и тип разметки.
func detectValueType(val string) (any, string) {
	if val == "<nil>" {
		return "", ""
	}
	// int
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i, "int"
	}

	// float
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f, "float"
	}

	// bool
	if b, err := strconv.ParseBool(val); err == nil {
		return b, "bool"
	}

	// time (несколько форматов)
	type timeLayout struct {
		layout string
		kind   string
	}
	layouts := []timeLayout{
		{time.RFC3339, "datetime"},                       // 2025-09-23T19:45:29+03:00
		{"2006-01-02T15:04:05.999999-07:00", "datetime"}, // 2025-09-23T19:45:29.754093+03:00
		{time.DateOnly, "date"},                          // 2025-09-23
		{time.DateTime, "datetime"},                      // 2025-09-23 19:45:29
		{"02.01.2006", "date"},                           // 23.09.2025
		{"02.01.2006 15:04:05", "datetime"},              // 23.09.2025 19:45:29
	}
	for _, l := range layouts {
		if t, err := time.Parse(l.layout, val); err == nil {
			return t, l.kind
		}
	}

	// строка по умолчанию
	return val, ""
}

func sanitizeSheetName(name string) string {
	// Удаляем или заменяем запрещённые символы
	replacer := strings.NewReplacer(
		":", "_",
		"\\", "_",
		"/", "_",
		"?", "_",
		"*", "_",
		"[", "_",
		"]", "_",
		" ", "_",
		"-", "_",
	)
	sanitized := replacer.Replace(name)

	// Обрезаем до 31 символа
	if len(sanitized) > 31 {
		sanitized = sanitized[:31]
	}

	// Удаляем ведущие пробелы
	sanitized = strings.TrimLeft(sanitized, " ")

	// Если начинается не с буквы или _, добавим префикс
	if sanitized == "" || !isValidSheetNameStart([]rune(sanitized)[0]) {
		sanitized = "Sheet_" + sanitized
	}

	return sanitized
}

func isValidSheetNameStart(b rune) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		b == '_' ||
		(b >= 'А' && b <= 'Я') || // Русские заглавные буквы
		(b >= 'а' && b <= 'я') // Русские строчные буквы
}

// getAutoWidth оценивает ширину колонки в символах.
func getAutoWidth(records [][]any, colIdx int) float64 {
	maxWidth := 10.0

	for _, row := range records {
		if colIdx < len(row) {
			width := float64(len([]rune(fmt.Sprint(row[colIdx])))) * 1.2 // запас
			if width > maxWidth {
				maxWidth = width
			}
		}
	}

	return maxWidth
}

func splitMeta(records []map[string]any) (
	meta map[string]any,
	data []map[string]any,
) {
	data = make([]map[string]any, 0)

	for _, record := range records {
		if m, ok := record["_meta"].(map[string]any); ok {
			meta = m

			continue
		}

		data = append(data, record)
	}

	return
}
