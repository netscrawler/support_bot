package httplib

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFile_SetsContentTypeFromExtension(t *testing.T) {
	rec := httptest.NewRecorder()

	File(rec, "report.csv", strings.NewReader("a,b,c"))

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("got Content-Type %q, want text/csv prefix", ct)
	}
	if rec.Body.String() != "a,b,c" {
		t.Fatalf("got body %q, want a,b,c", rec.Body.String())
	}
}

func TestFile_UnknownExtension_FallsBackToOctetStream(t *testing.T) {
	rec := httptest.NewRecorder()

	File(rec, "report.unknownext", strings.NewReader("data"))

	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("got Content-Type %q, want application/octet-stream", ct)
	}
}

func TestFile_SetsAttachmentDisposition(t *testing.T) {
	rec := httptest.NewRecorder()

	File(rec, "report.pdf", strings.NewReader("data"))

	cd := rec.Header().Get("Content-Disposition")
	if !strings.HasPrefix(cd, "attachment") || !strings.Contains(cd, `filename=report.pdf`) {
		t.Fatalf("got Content-Disposition %q, want attachment with filename=report.pdf", cd)
	}
}

func TestFile_WithContentType_OverridesDetectedType(t *testing.T) {
	rec := httptest.NewRecorder()

	File(
		rec,
		"report.csv",
		strings.NewReader("a,b"),
		WithContentType("text/csv; charset=windows-1251"),
	)

	if ct := rec.Header().Get("Content-Type"); ct != "text/csv; charset=windows-1251" {
		t.Fatalf("got Content-Type %q, want text/csv; charset=windows-1251", ct)
	}
}

func TestCSV_PrependsUTF8BOMAndSetsCharset(t *testing.T) {
	rec := httptest.NewRecorder()

	CSV(rec, "report.csv", strings.NewReader("a,b,c"))

	if ct := rec.Header().Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Fatalf("got Content-Type %q, want text/csv; charset=utf-8", ct)
	}

	want := "\ufeffa,b,c"
	if rec.Body.String() != want {
		t.Fatalf("got body %q, want %q", rec.Body.String(), want)
	}
}
