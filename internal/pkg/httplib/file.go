package httplib

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

type fileConfig struct {
	contentType string
}

type FileOption func(*fileConfig)

// WithContentType overrides the Content-Type that File would otherwise
// detect from the filename's extension.
func WithContentType(ct string) FileOption {
	return func(c *fileConfig) { c.contentType = ct }
}

// File writes r to w as a downloadable file response, deriving Content-Type
// from filename's extension (falling back to application/octet-stream)
// unless overridden via WithContentType.
func File(w http.ResponseWriter, filename string, r io.Reader, opts ...FileOption) {
	cfg := fileConfig{contentType: mime.TypeByExtension(filepath.Ext(filename))}
	if cfg.contentType == "" {
		cfg.contentType = "application/octet-stream"
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if disposition == "" {
		disposition = "attachment"
	}

	w.Header().Set("Content-Type", cfg.contentType)
	w.Header().Set("Content-Disposition", disposition)
	io.Copy(w, r)
}

// CSV writes r as a UTF-8 CSV file, prefixed with a byte-order mark so
// spreadsheet apps like Excel detect the encoding instead of misreading
// non-ASCII (e.g. Cyrillic) text.
func CSV(w http.ResponseWriter, filename string, r io.Reader) {
	File(
		w,
		filename,
		io.MultiReader(strings.NewReader("\ufeff"), r),
		WithContentType("text/csv; charset=utf-8"),
	)
}
