package xlsx

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"support_bot/internal/models"
	"support_bot/internal/pkg"

	"github.com/xuri/excelize/v2"
)

const (
	kindDate     = "date"
	kindDateTime = "datetime"
)

type Exporter struct {
	buf   map[string][]map[string]any
	order map[string][]string
	name  string
}

func New(
	data map[string][]map[string]any,
	name string,
	order map[string][]string,
) *Exporter {
	return &Exporter{
		buf:   data,
		order: order,
		name:  name,
	}
}

func (e *Exporter) Export() (*models.Data, error) {
	buf, err := e.createXlsxBook(e.buf)
	if err != nil {
		return nil, err
	}

	fd, err := models.NewFileData(buf, e.name+".xlsx")
	if err != nil {
		return nil, err
	}

	return &fd, nil
}

func (e *Exporter) createXlsxBook(
	dataMap map[string][]map[string]any,
) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	dateStyle, err := f.NewStyle(
		&excelize.Style{CustomNumFmt: new("dd.mm.yyyy")},
	)
	if err != nil {
		return nil, fmt.Errorf("создание стиля даты: %w", err)
	}

	dateTimeStyle, err := f.NewStyle(
		&excelize.Style{CustomNumFmt: new("dd.mm.yyyy hh:mm:ss")},
	)
	if err != nil {
		return nil, fmt.Errorf("создание стиля даты и времени: %w", err)
	}

	styles := map[string]int{
		kindDate:     dateStyle,
		kindDateTime: dateTimeStyle,
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

		if o, ok := e.order[unit]; ok {
			order = o
		} else {
			order = nil
		}

		sortedRecords := pkg.ConvertSortedRows(data, order)

		sheetName := sanitizeSheetName(unit)

		if _, err = f.NewSheet(sheetName); err != nil {
			return nil, fmt.Errorf("создание листа %s: %w", sheetName, err)
		}

		for rowIdx, row := range sortedRecords {
			for colIdx, val := range row {
				cell, cErr := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
				if cErr != nil {
					return nil, fmt.Errorf("имя ячейки: %w", cErr)
				}

				v, kind := detectValueType(fmt.Sprint(val))
				if err = f.SetCellValue(sheetName, cell, v); err != nil {
					return nil, fmt.Errorf("запись ячейки %s: %w", cell, err)
				}

				if styleID, ok := styles[kind]; ok {
					err = f.SetCellStyle(sheetName, cell, cell, styleID)
					if err != nil {
						return nil, fmt.Errorf("стиль ячейки %s: %w", cell, err)
					}
				}
			}
		}

		// Добавление таблицы (с фильтрацией)
		startCell, cErr := excelize.CoordinatesToCellName(1, 1)
		if cErr != nil {
			return nil, fmt.Errorf("имя ячейки: %w", cErr)
		}

		endCell, cErr := excelize.CoordinatesToCellName(
			len(sortedRecords[0]),
			len(sortedRecords),
		)
		if cErr != nil {
			return nil, fmt.Errorf("имя ячейки: %w", cErr)
		}

		tableRange := fmt.Sprintf("%s:%s", startCell, endCell)

		a := true

		err = f.AddTable(sheetName, &excelize.Table{
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
			return nil, fmt.Errorf(
				"ошибка создания таблицы на листе %s: %w",
				sheetName,
				err,
			)
		}

		// Автоширина колонок
		for colIdx := range sortedRecords[0] {
			colLetter, cErr := excelize.ColumnNumberToName(colIdx + 1)
			if cErr != nil {
				return nil, fmt.Errorf("имя колонки: %w", cErr)
			}

			colRange := colLetter + ":" + colLetter

			if err = f.SetColWidth(
				sheetName,
				colRange,
				colRange,
				getAutoWidth(sortedRecords, colIdx),
			); err != nil {
				return nil, fmt.Errorf("ширина колонки %s: %w", colLetter, err)
			}
		}
	}

	if err = f.DeleteSheet("Sheet1"); err != nil {
		return nil, fmt.Errorf("удаление листа Sheet1: %w", err)
	}

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
		{
			time.RFC3339,
			kindDateTime,
		}, // 2025-09-23T19:45:29+03:00
		{
			"2006-01-02T15:04:05.999999-07:00",
			kindDateTime,
		}, // 2025-09-23T19:45:29.754093+03:00
		{time.DateOnly, kindDate},             // 2025-09-23
		{time.DateTime, kindDateTime},         // 2025-09-23 19:45:29
		{"02.01.2006", kindDate},              // 23.09.2025
		{"02.01.2006 15:04:05", kindDateTime}, // 23.09.2025 19:45:29
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
	first, _ := utf8.DecodeRuneInString(sanitized)
	if sanitized == "" || !isValidSheetNameStart(first) {
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
			width := float64(
				len([]rune(fmt.Sprint(row[colIdx]))),
			) * 1.2 // запас
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

	return meta,
		data
}
