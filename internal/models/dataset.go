package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
)

type Dataset map[string][]map[string]any

// metaSheet — служебный лист (время генерации и т.п.), в хэш не входит.
const (
	metaSheet = "_meta"
)

// Hash — sha256 содержимого без листа _meta; порядок ключей строки не важен, порядок строк важен.
// Пустая строка — датасет не маршалится (дедупликация для этого запуска не срабатывает).
func (d Dataset) Hash() string {
	c := maps.Clone(d)
	delete(c, metaSheet)

	b, err := json.Marshal(c)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(b)

	return hex.EncodeToString(sum[:])
}
