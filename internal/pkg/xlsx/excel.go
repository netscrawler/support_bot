package xlsx

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func CreateXlsxBook(
	dataMap map[string][][]string,
) (*bytes.Buffer, error) {
	f := excelize.NewFile()

	for unit, records := range dataMap {
		if len(records) == 0 {
			continue
		}

		sheetName := sanitizeSheetName(unit)
		f.NewSheet(sheetName)

		for rowIdx, row := range records {
			for colIdx, val := range row {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				f.SetCellValue(sheetName, cell, detectValueType(val))
			}
		}

		// Добавление таблицы (с фильтрацией)
		startCell, _ := excelize.CoordinatesToCellName(1, 1)
		endCell, _ := excelize.CoordinatesToCellName(len(records[0]), len(records))
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
		for colIdx := range records[0] {
			colLetter, _ := excelize.ColumnNumberToName(colIdx + 1)
			colRange := colLetter + ":" + colLetter
			f.SetColWidth(sheetName, colRange, colRange, getAutoWidth(records, colIdx))
		}
	}

	f.DeleteSheet("Sheet1")

	return f.WriteToBuffer()
}

// detectValueType определяет тип значения по строке и возвращает подходящий тип.
func detectValueType(val string) any {
	// int
	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		return i
	}

	// float
	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return f
	}

	// bool
	if b, err := strconv.ParseBool(val); err == nil {
		return b
	}

	// time (несколько форматов)
	layouts := []string{
		time.RFC3339,                       // 2025-09-23T19:45:29+03:00
		"2006-01-02T15:04:05.999999-07:00", // 2025-09-23T19:45:29.754093+03:00
		time.DateOnly,                      // 2025-09-23
		time.DateTime,                      // 2025-09-23 19:45:29
		"02.01.2006",                       // 23.09.2025
		"02.01.2006 15:04:05",              // 23.09.2025 19:45:29
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, val); err == nil {
			return t
		}
	}

	// строка по умолчанию
	return val
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
func getAutoWidth(records [][]string, colIdx int) float64 {
	max := 10.0

	for _, row := range records {
		if colIdx < len(row) {
			width := float64(len([]rune(row[colIdx]))) * 1.2 // запас
			if width > max {
				max = width
			}
		}
	}

	return max
}
