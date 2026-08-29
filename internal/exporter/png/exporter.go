package png

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
	"support_bot/internal/models"
	"support_bot/internal/pkg"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

type Exporter struct{}

func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	// Если есть Layout — используем его для рендеринга табличных блоков в PNG
	if format.Layout != nil {
		return e.exportFromLayout(data, *format.Layout, format)
	}

	// Legacy режим
	var err error

	var id []models.Data

	for k, v := range data {
		_, pureData := splitMeta(v)

		var order []string
		if o, ok := format.Order[k]; ok {
			order = o
		} else {
			order = nil
		}

		mtx := pkg.ConvertSortedRows(pureData, order)

		img, gErr := createImageFromMatrix(mtx, &k)
		if gErr != nil {
			err = errors.Join(err, gErr)

			continue
		}

		dt, eErr := models.NewImageData(img, *format.FileName+"_"+k+".png")
		if eErr != nil {
			err = errors.Join(err, eErr)
		}

		id = append(id, dt)
	}

	return id, nil
}

func (e Exporter) exportFromLayout(data models.Dataset, layout models.Layout, format models.Export) ([]models.Data, error) {
	var outputs []models.Data

	for _, block := range layout.Blocks {
		if block.Type != models.BlockTypeTable {
			continue
		}

		if block.Table == nil || block.Dataset == "" {
			continue
		}

		rows, ok := data[block.Dataset]
		if !ok || len(rows) == 0 {
			continue
		}

		// Формируем порядок колонок из блока
		var ordering []string
		for _, col := range block.Table.Columns {
			ordering = append(ordering, col.Field)
		}

		mtx := pkg.ConvertSortedRows(rows, ordering)

		title := block.ID
		img, err := createImageFromMatrix(mtx, &title)
		if err != nil {
			return nil, fmt.Errorf("create image for block %q: %w", block.ID, err)
		}

		filename := *format.FileName + "_" + block.ID + ".png"
		dt, err := models.NewImageData(img, filename)
		if err != nil {
			return nil, fmt.Errorf("create image data for block %q: %w", block.ID, err)
		}

		outputs = append(outputs, dt)
	}

	if len(outputs) == 0 {
		return nil, errors.New("no table blocks found in layout")
	}

	return outputs, nil
}

const (
	headerHEX = "#808080"
	rowHEX    = "#d5d5d5"
)

func createImageFromMatrix(data [][]any, title *string) (*bytes.Buffer, error) {
	faceNormal, err := pkg.GetFontFaceNormal(18)
	if err != nil {
		return nil, err
	}

	imageFromMatrix := generateTableImageFromMatrix(data, faceNormal, 18.0, 6, 1)

	titleFont, err := pkg.GetFontFaceBold(24)
	if err != nil {
		return nil, err
	}

	if title != nil {
		imageFromMatrix, err = addTitleAboveImage(imageFromMatrix, *title, titleFont, 10)
		if err != nil {
			return nil, err
		}
	}

	buf := new(bytes.Buffer)

	err = png.Encode(buf, imageFromMatrix)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// generateTableImageFromMatrix генерирует изображение таблицы из CSV-данных (records [][]string)
// font — ttf шрифт
// fontSize — размер шрифта
// padding — внутренний отступ внутри ячеек (с каждой стороны)
// borderWidth — ширина рамок таблицы.
func generateTableImageFromMatrix(
	records [][]any,
	font font.Face,
	fontSize float64,
	padding float64,
	borderWidth float64,
) image.Image {
	// Кол-во столбцов по первой строке
	colsCount := 0

	for _, row := range records {
		if len(row) > colsCount {
			colsCount = len(row)
		}
	}

	// Используем face в gg.Context
	dc := gg.NewContext(10, 10)
	dc.SetFontFace(font)

	// Минимальная ширина колонки (если надо, можно задать)
	minColWidth := 50.0

	// 1. Рассчитать ширину столбцов
	colWidths := make([]float64, colsCount)

	// Предварительно положим ширину колонки как ширина заголовка (первой строки)
	for col := range colsCount {
		if len(records[0]) > col {
			w, _ := dc.MeasureString(fmt.Sprint(records[0][col]))
			colWidths[col] = w
		} else {
			colWidths[col] = minColWidth
		}
	}

	// Для остальных строк учитываем максимальную ширину слов (без переноса)
	for _, row := range records {
		for col := range colsCount {
			cellText := ""

			if len(row) > col {
				cellText = fmt.Sprint(row[col])
			}

			// Без переноса измерим полную ширину
			w, _ := dc.MeasureString(cellText)
			if w > colWidths[col] {
				colWidths[col] = w
			}
		}
	}

	// Добавим padding к ширинам
	for i := range colWidths {
		colWidths[i] += 2 * padding
		if colWidths[i] < minColWidth {
			colWidths[i] = minColWidth
		}
	}

	// 2. Рассчитать высоту каждой строки с учётом переноса текста по ширине колонки
	rowHeights := make([]float64, len(records))
	lineHeight := fontSize * 1.4

	for i, row := range records {
		maxHeight := 0.0

		for col := range colsCount {
			cellText := ""

			if len(row) > col {
				cellText = fmt.Sprint(row[col])
			}

			maxTextWidth := colWidths[col] - 2*padding
			lines := wrapText(dc, cellText, maxTextWidth)

			h := float64(len(lines)) * lineHeight
			if h > maxHeight {
				maxHeight = h
			}
		}
		// Минимальная высота строки равна высоте строки шрифта + padding сверху и снизу
		minHeight := lineHeight + 2*padding
		if maxHeight < minHeight {
			maxHeight = minHeight
		}

		rowHeights[i] = maxHeight
	}

	// 3. Рассчитать общую ширину и высоту изображения
	totalWidth := borderWidth // левая рамка

	for _, w := range colWidths {
		totalWidth += w + borderWidth
	}

	totalHeight := borderWidth // верхняя рамка

	for _, h := range rowHeights {
		totalHeight += h + borderWidth
	}

	// 4. Создать изображение
	imgWidth := int(math.Ceil(totalWidth))
	imgHeight := int(math.Ceil(totalHeight))

	dc = gg.NewContext(imgWidth, imgHeight)
	dc.SetColor(color.White)
	dc.Clear()
	dc.SetFontFace(font)

	// 5. Нарисовать таблицу с рамками и текстом
	y := borderWidth

	for rowIdx, row := range records {
		x := borderWidth
		rowHeight := rowHeights[rowIdx]

		for col := range colsCount {
			cellWidth := colWidths[col]

			switch {
			case rowIdx == 0:
				dc.SetHexColor(headerHEX) // светло-серый
			case rowIdx%2 == 0:
				dc.SetHexColor(rowHEX) // светло-серый
			case rowIdx%2 != 0:
				dc.SetColor(color.White)
			}
			// dc.SetColor(color.White)
			dc.DrawRectangle(x, y, cellWidth, rowHeight)
			dc.Fill()

			// Рисуем рамку ячейки
			dc.SetColor(color.Black)
			dc.SetLineWidth(borderWidth)
			dc.DrawRectangle(x, y, cellWidth, rowHeight)
			dc.Stroke()

			// Текст в ячейке
			cellText := ""

			if len(row) > col {
				cellText = fmt.Sprint(row[col])
				if row[col] == nil {
					cellText = ""
				}
			}

			lines := wrapText(dc, cellText, cellWidth-2*padding)

			// Вертикальное выравнивание - отступ сверху + размер шрифта
			startY := y + padding + fontSize

			for i, line := range lines {
				// Горизонтальное выравнивание — слева с padding
				dc.SetColor(color.Black)
				dc.DrawStringAnchored(line, x+padding, startY+float64(i)*lineHeight, 0, 0)
			}

			x += cellWidth + borderWidth
		}

		y += rowHeight + borderWidth
	}

	return dc.Image()
}

func addTitleAboveImage(
	img image.Image,
	title string,
	face font.Face,
	padding float64,
) (image.Image, error) {
	// Ширина и высота оригинального изображения
	imgW := img.Bounds().Dx()
	imgH := img.Bounds().Dy()

	// Контекст для измерения текста
	dc := gg.NewContext(imgW, 100) // высота 100 — временно
	dc.SetFontFace(face)

	_, textH := dc.MeasureString(title)

	// Высота новой картинки = заголовок + padding + исходная высота
	titleHeight := int(textH + 2*padding)
	totalHeight := imgH + titleHeight

	// Создаем новое изображение
	newImg := image.NewRGBA(image.Rect(0, 0, imgW, totalHeight))
	draw.Draw(newImg, newImg.Bounds(), image.White, image.Point{}, draw.Src)

	// Отрисовываем заголовок
	dc = gg.NewContextForRGBA(newImg)
	dc.SetFontFace(face)
	dc.SetRGB(0, 0, 0) // черный цвет
	dc.DrawStringAnchored(title, float64(imgW)/2, padding+textH/2, 0.5, 0.3)

	// Отрисовываем оригинальное изображение ниже
	dc.DrawImage(img, 0, titleHeight)

	return dc.Image(), nil
}

// wrapText разбивает текст на строки, чтобы каждая строка умещалась в maxWidth.
func wrapText(dc *gg.Context, text string, maxWidth float64) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var (
		lines []string
		line  string
	)

	for _, word := range words {
		test := line
		if line != "" {
			test += " "
		}

		test += word

		if w, _ := dc.MeasureString(test); w > maxWidth {
			if line != "" {
				lines = append(lines, line)
			}

			line = word
		} else {
			line = test
		}
	}

	if line != "" {
		lines = append(lines, line)
	}

	return lines
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
