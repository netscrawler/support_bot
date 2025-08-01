package png

import (
	"bytes"
	"image/png"
	"support_bot/internal/pkg"
)

func CreateImageFromMatrix(data [][]string, name string, title string) (*bytes.Buffer, error) {
	font, err := pkg.GetFontFaceNormal(18)
	if err != nil {
		return nil, err
	}

	image := GenerateTableImageFromMatrix(data, font, 18.0, 6, 1)

	titleFont, err := pkg.GetFontFaceBold(24)
	if err != nil {
		return nil, err
	}

	image, err = AddTitleAboveImage(image, title, titleFont, 24, 10)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	err = png.Encode(buf, image)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
