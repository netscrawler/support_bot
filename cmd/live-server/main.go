package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	tmplTXT "text/template"
	"time"

	"support_bot/internal/collector"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/pkg/text"

	models "support_bot/internal/models/report"

	"github.com/Masterminds/sprig/v3"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

type config struct {
	Cards           []models.Card `json:"cards"`
	MetabaseBaseURL string        `json:"metabase"`
}

func main() {
	log := slog.Default()

	log.Info("starting live server")

	cwd, err := os.Getwd()
	if err != nil {
		log.Error("error check current dir", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("loading config", slog.Any("dir", cwd))

	cfgPath := filepath.Join(cwd, "config.json")

	rawCfg, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Error("error load config from", slog.Any("path", cfgPath), slog.Any("error", err))
		os.Exit(1)
	}

	cfg, err := UnmarshalFor[config](rawCfg)
	if err != nil {
		log.Error("error unmarshall config", slog.Any("error", err))
		os.Exit(1)
	}

	fsCwd := os.DirFS(cwd)

	ctx := context.Background()

	log.Info("loading templates")

	templatesHTML := loadHTMLTemplates(fsCwd, log)
	templatesText := loadTXTTemplates(fsCwd, log)

	if len(templatesHTML) == 0 && len(templatesText) == 0 {
		log.Error("templates not found")
		os.Exit(1)
	}

	clientMap := map[string]map[*websocket.Conn]struct{}{}

	for _, tmpl := range append(templatesHTML, templatesText...) {
		clientMap[tmpl] = map[*websocket.Conn]struct{}{}
	}

	wsH := wsHandler{
		upg:     websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients: clientMap,
		mu:      sync.Mutex{},
	}

	slog.Info("Starting watcher")
	if err := watchTemplateDir(
		wsH.notifyReload,
		append(templatesText, templatesHTML...)...); err != nil {
		slog.Error("error watch for templates", slog.Any("error", err))
	}

	clct := collector.NewCollector(4, metabase.New(cfg.MetabaseBaseURL), slog.Default())

	slog.Info("collecting data for cards")

	data, err := clct.Collect(ctx, cfg.Cards...)
	if err != nil {
		log.Error("error collecting data", slog.Any("error", err))
		os.Exit(1)
	}

	h := handler{
		data:      data,
		templates: sliceToMap(append(templatesHTML, templatesText...)),
	}

	http.Handle(
		"/html/{template}",
		logMiddleware(h.existMiddleware(liveReloadMiddleware(http.HandlerFunc(h.HandleHTML)))),
	)
	http.Handle(
		"/text/{template}",
		logMiddleware(h.existMiddleware(liveReloadMiddleware(http.HandlerFunc(h.HandleTXT)))),
	)

	http.Handle("/ws/{template}", h.existMiddleware(http.HandlerFunc(wsH.wsHandle)))

	srv := &http.Server{
		Addr:              "localhost:8080",
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		slog.Info(fmt.Sprintf("server started http://%s", srv.Addr))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		slog.Info("received stop signal", "signal", sig)

	case err := <-errCh:
		slog.Error("server error", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "error", err)
	} else {
		slog.Info("server shutdown complete")
	}
}

func sliceToMap(sl []string) map[string]struct{} {
	m := make(map[string]struct{}, len(sl))
	for _, s := range sl {
		m[s] = struct{}{}
	}

	return m
}

type handler struct {
	templates map[string]struct{}

	data map[string][]map[string]any
}

func (h *handler) HandleHTML(w http.ResponseWriter, r *http.Request) {
	templ := r.PathValue("template")

	allFuncs := sprig.HtmlFuncMap()
	maps.Copy(allFuncs, text.FuncMap)

	t, err := template.New("").
		Funcs(allFuncs).
		ParseFiles(templ)
	if err != nil {
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintf(w, "Error while parse or load template file : %s", err.Error())

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = t.ExecuteTemplate(w, templ, h.data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error while execute template : %s", err.Error())
	}
}

// html.
const textWrap = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>{{.TemplateName}}</title>
  </head>
  <body>
    <div style="white-space: pre-line">{{.Content}}</div>
  </body>
</html>
`

func (h *handler) HandleTXT(w http.ResponseWriter, r *http.Request) {
	templ := r.PathValue("template")

	allFuncs := sprig.TxtFuncMap()
	maps.Copy(allFuncs, text.FuncMap)

	t, err := tmplTXT.New("").
		Funcs(allFuncs).
		ParseFiles(templ)
	if err != nil {
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintf(w, "Error while parse or load template file : %s", err.Error())

		return
	}

	var buf bytes.Buffer

	err = t.ExecuteTemplate(&buf, templ, h.data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error while execute template : %s", err.Error())
	}

	dt := struct {
		TemplateName string
		Content      template.HTML
	}{
		TemplateName: templ,
		Content:      template.HTML(buf.String()),
	}

	tW, err := template.New("text_wrap").Parse(textWrap)
	if err != nil {
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintf(w, "Error while parse or load template file : %s", err.Error())

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = tW.ExecuteTemplate(w, "text_wrap", dt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error while execute template : %s", err.Error())
	}
}

func (h *handler) existMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templ := r.PathValue("template")
		if _, ok := h.templates[templ]; !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found template: %s", templ)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := &responseCapture{ResponseWriter: w}
		slog.InfoContext(r.Context(), "getting request", slog.Any("path", r.URL.String()))

		next.ServeHTTP(rc, r)

		buf := rc.buf.String()

		if rc.statusCode != 0 {
			slog.Error(
				"errror processing request",
				slog.Any("path", r.URL.String()),
				slog.Any("code", rc.statusCode),
				slog.Any("response", buf),
			)
			w.WriteHeader(rc.statusCode)
		}
		w.Write([]byte(buf))
	})
}

type responseCapture struct {
	http.ResponseWriter

	buf        bytes.Buffer
	statusCode int
}

func (r *responseCapture) WriteHeader(code int) {
	r.statusCode = code
}

func (r *responseCapture) Write(b []byte) (int, error) {
	return r.buf.Write(b)
}

// html.
const liveReloadScript = `
<script>
  (() => {
    const ws = new WebSocket("ws://localhost:8080/ws/%s");
    ws.onmessage = (e) => {
      if (e.data === "reload") {
        location.reload();
      }
    };
  })();
</script>
`

func liveReloadMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := &responseCapture{ResponseWriter: w}

		next.ServeHTTP(rc, r)

		body := rc.buf.Bytes()

		ct := w.Header().Get("Content-Type")
		if strings.Contains(ct, "text/html") {
			body = bytes.Replace(
				body,
				[]byte("</body>"),
				[]byte(fmt.Sprintf(liveReloadScript, r.PathValue("template"))+"</body>"),
				1,
			)
		}

		if rc.statusCode != 0 {
			w.WriteHeader(rc.statusCode)
		}

		w.Write(body)
	})
}

type wsHandler struct {
	upg websocket.Upgrader

	clients map[string]map[*websocket.Conn]struct{}

	mu sync.Mutex
}

func (h *wsHandler) wsHandle(w http.ResponseWriter, r *http.Request) {
	tmpl := r.PathValue("template")

	conn, err := h.upg.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	h.mu.Lock()

	_, ok := h.clients[tmpl]
	if !ok {
		return
	}

	h.clients[tmpl][conn] = struct{}{}
	h.mu.Unlock()

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients[tmpl], conn)
			h.mu.Unlock()
			conn.Close()
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (h *wsHandler) notifyReload(tmpl string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	cl, ok := h.clients[tmpl]
	if !ok {
		return
	}

	for c := range cl {
		_ = c.WriteMessage(websocket.TextMessage, []byte("reload"))
	}
}

func watchTemplateDir(onEvent func(tmpl string), templates ...string) error {
	var watchErr error

	for _, t := range templates {
		err := watchTemplate(t, onEvent)
		if err != nil {
			watchErr = errors.Join(watchErr, err)
		}
	}

	return watchErr
}

func watchTemplate(path string, onEvent func(tmpl string)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	file := filepath.Base(path)

	var debounce *time.Timer

	go func() {
		defer watcher.Close()

		for {
			select {
			case ev := <-watcher.Events:
				if filepath.Base(ev.Name) != file {
					continue
				}

				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
					slog.Info(
						"getting event",
						slog.Any("template", path),
						slog.Any("event", ev.Name),
						slog.Any("event_op", ev.Op.String()),
					)
					if debounce != nil {
						debounce.Stop()
					}

					debounce = time.AfterFunc(200*time.Millisecond, func() {
						onEvent(path)
					})
				}

			case err := <-watcher.Errors:
				slog.Error("watcher error", "error", err)
			}
		}
	}()

	return watcher.Add(dir)
}

func loadTXTTemplates(f fs.FS, log *slog.Logger) []string {
	var templates []string

	match, err := fs.Glob(f, "*.txt")
	if err != nil {
		log.Info("error check glob", slog.Any("error", err))

		return templates
	}

	templates = append(templates, match...)

	if len(templates) > 0 {
		log.Info(
			"loaded text templates",
			slog.Any("count", len(templates)),
			slog.Any("templates", templates),
		)
	}

	return templates
}

func loadHTMLTemplates(f fs.FS, log *slog.Logger) []string {
	var templates []string

	patterns := []string{"*.html", "*.gotmpl", "*.tmpl"}
	for _, p := range patterns {
		match, err := fs.Glob(f, p)
		if err != nil {
			log.Info("error check glob", slog.Any("error", err))

			continue
		}

		templates = append(templates, match...)
	}

	if len(templates) > 0 {
		log.Info(
			"loaded html templates",
			slog.Any("count", len(templates)),
			slog.Any("templates", templates),
		)
	}

	return templates
}

func UnmarshalFor[V any](data []byte) (V, error) {
	var v V

	err := json.Unmarshal(data, &v)

	return v, err
}
