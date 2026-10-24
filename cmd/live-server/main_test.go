package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLiveReloadEscapesTemplatePath(t *testing.T) {
	t.Parallel()

	name := `x"</script><script>alert(1)</script>`
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.SetPathValue("template", name)

	rec := httptest.NewRecorder()
	handler := liveReloadMiddleware(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			_, err := w.Write([]byte("<body>ok</body>"))
			if err != nil {
				t.Errorf("write: %v", err)
			}
		}),
	)
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, name) ||
		!strings.Contains(body, url.PathEscape(name)) {
		t.Fatalf("template path was not escaped: %q", body)
	}
}

func TestTextWrapEscapesContent(t *testing.T) {
	t.Parallel()

	tmpl, err := template.New("text_wrap").Parse(textWrap)
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder

	if err := tmpl.Execute(
		&out,
		struct{ TemplateName, Content string }{
			Content: "<script>alert(1)</script>",
		},
	); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out.String(), "<script>alert(1)</script>") {
		t.Fatalf("content was not escaped: %q", out.String())
	}
}
