package xlsx

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// CreateXlsxBook сохраняет все records по юнитам в одну XLSX-книгу, на разные листы.
// Аргументы:
// - dataMap: map, где ключ — unit, значение — CSV данные ([][]string).
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

				if colIdx == 2 {
					valInt, err := strconv.ParseInt(val, 10, 64)
					if err == nil {
						f.SetCellValue(sheetName, cell, valInt)

						continue
					}
				}

				f.SetCellValue(sheetName, cell, val)
			}
		}

		// Добавление таблицы (со включенной фильтрацией)
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
