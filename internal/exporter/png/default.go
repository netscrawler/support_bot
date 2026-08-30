//go:build !chromium

package png

import (
	"bytes"
	"context"
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

	layoutpkg "support_bot/internal/layout"
)

// HtmlExporter matches the chromium build's constructor signature; the pure-Go
// rasterizer does not need it.
type HtmlExporter interface {
	Export(data models.Dataset, exp models.Export) ([]models.Data, error)
}

type Exporter struct{}

// New returns the pure-Go table rasterizer. shellPath and h are accepted for
// parity with the chromium build and ignored.
func New(_ string, _ HtmlExporter) *Exporter {
	return &Exporter{}
}

// Format implements render.Renderer.
func (e Exporter) Format() string { return models.ReportFormatPng }

// Geometry implements render.Renderer: the gg rasterizer draws one image per
// table block, so blocks stay at their preferred size on a single page.
func (e Exporter) Geometry(_ models.PageConfig) layoutpkg.PageGeometry {
	return layoutpkg.PageGeometry{Rows: 0}
}

// Export renders one export. A declarative layout flows through the shared
// Resolve/Pack/Render pipeline and yields one PNG per table block; otherwise
// the legacy "one image per dataset key" behaviour is kept.
func (e Exporter) Export(data models.Dataset, format models.Export) ([]models.Data, error) {
	if format.Layout != nil {
		return e.exportLayout(data, format)
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

// Render implements render.Renderer: one rasterised PNG per resolved table
// block, titled with the block id.
func (e Exporter) Render(
	_ context.Context,
	_ models.Dataset,
	_ layoutpkg.ResolvedDocument,
	pages []layoutpkg.Page,
	format models.Export,
) ([]models.Data, error) {
	name := "report"
	if format.FileName != nil {
		name = *format.FileName
	}

	var outputs []models.Data

	for _, page := range pages {
		for _, pb := range page.Blocks {
			if pb.Block.Type != models.BlockTypeTable || pb.Block.Table == nil {
				continue
			}

			ordering := make([]string, 0, len(pb.Block.Table.Columns))
			for _, col := range pb.Block.Table.Columns {
				ordering = append(ordering, col.Field)
			}

			end := min(pb.RowOffset+pb.RowCount, len(pb.Rows))
			mtx := pkg.ConvertSortedRows(pb.Rows[pb.RowOffset:end], ordering)

			blockID := pb.Block.ID

			img, err := createImageFromMatrix(mtx, &blockID)
			if err != nil {
				return nil, fmt.Errorf("png: image for block %q: %w", blockID, err)
			}

			dt, err := models.NewImageData(img, name+"_"+blockID+".png")
			if err != nil {
				return nil, fmt.Errorf("png: image data for block %q: %w", blockID, err)
			}

			outputs = append(outputs, dt)
		}
	}

	if len(outputs) == 0 {
		return nil, errors.New("png: layout has no table blocks")
	}

	return outputs, nil
}

func (e Exporter) exportLayout(data models.Dataset, format models.Export) ([]models.Data, error) {
	l := layoutpkg.LayoutFromExport(format)

	if err := layoutpkg.Validate(l, layoutpkg.SortedDatasetKeys(data)); err != nil {
		return nil, fmt.Errorf("png: invalid layout: %w", err)
	}

	doc, err := layoutpkg.Resolve(l, data)
	if err != nil {
		return nil, fmt.Errorf("png: resolve layout: %w", err)
	}

	pages, err := layoutpkg.Pack(doc, e.Geometry(doc.Page))
	if err != nil {
		return nil, fmt.Errorf("png: pack layout: %w", err)
	}

	return e.Render(context.Background(), data, doc, pages, format)
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
