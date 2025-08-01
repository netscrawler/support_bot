package pkg

import (
	"embed"
	"support_bot/assets"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

func GetFontFaceBold(fontSize float64) (font.Face, error) {
	return GetFontFaceFromEmbedFS(assets.FontFS, "fonts/bold.ttf", fontSize)
}

func GetFontFaceNormal(fontSize float64) (font.Face, error) {
	return GetFontFaceFromEmbedFS(assets.FontFS, "fonts/font.ttf", fontSize)
}

func GetFontFaceFromEmbedFS(fs embed.FS, path string, fontSize float64) (font.Face, error) {
	// Чтение шрифта из embed.FS
	fontBytes, err := fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fnt, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	return face, nil
}
