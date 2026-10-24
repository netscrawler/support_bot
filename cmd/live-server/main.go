package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	tmplTXT "text/template"
	"time"

	"support_bot/internal/collector"
	"support_bot/internal/collector/appmetrica"
	"support_bot/internal/collector/grafana"
	"support_bot/internal/collector/httpcoll"
	"support_bot/internal/collector/jira"
	"support_bot/internal/collector/metabase"
	"support_bot/internal/collector/postgres"
	"support_bot/internal/collector/prometheus"
	"support_bot/internal/collector/zabbix"
	"support_bot/internal/models"
	"support_bot/internal/pkg/text"

	"github.com/Masterminds/sprig/v3"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var (
	Version   = "v0.0.0"
	commit    = "unknown"
	buildTime = "unknown"
)

var (
	setModeVer          = false
	setModeHelp         = false
	setModeCreateConfig = false

	configPath    = ""
	templatesPath = ""
)

func modeStart() {
	flag.BoolVar(&setModeVer, "v", false, "Версия приложения")
	flag.BoolVar(&setModeHelp, "h", false, "Помощь")
	flag.StringVar(&configPath, "config", "", "Путь к файлу конфигурации")
	flag.StringVar(
		&templatesPath,
		"templates",
		"",
		"Путь к директории с шаблонами",
	)
	flag.BoolVar(
		&setModeCreateConfig,
		"example-config",
		false,
		"Сгенерировать пример файла конфигурации",
	)
	flag.Parse()

	if setModeVer {
		version()
	}

	if setModeHelp {
		help()
	}

	if setModeCreateConfig {
		configDef()
	}
}

func writeOutput(value string) {
	if _, err := os.Stdout.WriteString(value); err != nil {
		slog.Error("write stdout", slog.Any("error", err))
		os.Exit(1)
	}
}

func version() {
	writeOutput(fmt.Sprintf(
		"Version: %s\nCommit: %s\nBuildTime: %s\nRuntime: %s",
		Version, commit, buildTime, runtime.Version(),
	))
	os.Exit(0)
}

func configDef() {
	raw, err := json.MarshalIndent(config{
		Cards: []models.Card{{
			CardUUID: "288ac24c-73e2-4fac-b386-2b5cc5cb5934",
			Title:    "Example card",
		}, {
			CardUUID: "a7dcc5d9-d3f1-4066-853b-f8f21f4b5134",
			Title:    "Example card 2",
		}},
		MetabaseBaseURL: "https://metabase.example.com",
		Jira: jira.Config{
			AuthToken: "token",
			JiraHost:  "host.jira.com",
			Timeout:   0,
		},
		AppMetrica: appmetrica.Config{
			OAuthToken: "oauth_token",
			Timeout:    0,
		},
	}, " ", "  ")
	if err != nil {
		slog.Error("unable create config", slog.Any("error", err))
		os.Exit(1)
	}

	writeOutput(string(raw) + "\n")
	os.Exit(0)
}

func help() {
	writeOutput("Доступные флаги и их описания:\n")
	flag.PrintDefaults()
	os.Exit(0)
}

type config struct {
	Cards           []models.Card     `json:"cards"`
	MetabaseBaseURL string            `json:"metabase"`
	Jira            jira.Config       `json:"jira"`
	AppMetrica      appmetrica.Config `json:"appmetrica"`

	Databases  map[string]postgres.Config          `json:"databases"`
	Prometheus map[string]prometheus.Config        `json:"prometheus"`
	Grafana    map[string]prometheus.GrafanaConfig `json:"grafana"`
	Zabbix     map[string]zabbix.Config            `json:"zabbix"`

	HTTPSources map[string]httpcoll.Config `json:"http_sources"`
}

func main() {
	modeStart()

	log := slog.Default()

	log.Info("starting live server")

	cwd, err := os.Getwd()
	if err != nil {
		log.Error("error check current dir", slog.Any("error", err))
		os.Exit(1)
	}

	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = filepath.Join(cwd, "config.json")
	}

	log.Info("loading config", slog.Any("dir", cfgPath))

	rawCfg, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Error(
			"error load config from",
			slog.Any("path", cfgPath),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	cfg, err := unmarshalFor[config](rawCfg)
	if err != nil {
		log.Error("error unmarshall config", slog.Any("error", err))
		os.Exit(1)
	}

	var fsCwd fs.FS

	if templatesPath != "" {
		fsCwd = os.DirFS(templatesPath)
	} else {
		fsCwd = os.DirFS(cwd)
	}

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
		upg: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		clients: clientMap,
		mu:      sync.Mutex{},
	}

	slog.Info("Starting watcher")

	if err = watchTemplateDir(
		wsH.notifyReload,
		append(templatesText, templatesHTML...)...); err != nil {
		slog.Error("error watch for templates", slog.Any("error", err))
	}

	prom, err := prometheus.New(cfg.Prometheus, cfg.Grafana, log)
	if err != nil {
		log.Error("error create prometheus collector", slog.Any("error", err))
		os.Exit(1)
	}

	zbx, err := zabbix.New(cfg.Zabbix, log)
	if err != nil {
		log.Error("error create zabbix collector", slog.Any("error", err))
		os.Exit(1)
	}

	// Секреты из БД в live-server не поддерживаются: secrets = nil.
	httpColl, err := httpcoll.New(cfg.HTTPSources, nil, log)
	if err != nil {
		log.Error("error create http collector", slog.Any("error", err))
		os.Exit(1)
	}

	pg, err := postgres.New(ctx, cfg.Databases, log)
	if err != nil {
		log.Error("error create postgres collector", slog.Any("error", err))
		os.Exit(1)
	}

	clct := collector.NewCollector(
		4,
		collector.Fetchers{
			models.CollectTypeMetabase: collector.IgnoreQuery(
				metabase.New(cfg.MetabaseBaseURL),
			),
			models.CollectTypeAppMetrica: collector.IgnoreQuery(
				appmetrica.NewCollector(&cfg.AppMetrica, log),
			),
			models.CollectTypeJIRA: collector.IgnoreQuery(
				jira.New(cfg.Jira),
			),
			models.CollectTypePostgres:      pg,
			models.CollectTypePrometheus:    prom,
			models.CollectTypeGrafanaAlerts: grafana.New(cfg.Grafana, log),
			models.CollectTypeZabbix:        zbx,
			models.CollectTypeHTTP:          httpColl,
		},
		slog.Default(),
	)

	slog.Info("collecting data for cards")

	data, err := clct.Collect(ctx, cfg.Cards...)
	if err != nil {
		log.Error("error collecting data", slog.Any("error", err))
		pg.Close()
		os.Exit(1)
	}

	defer pg.Close()

	h := handler{
		data:      data,
		templates: sliceToMap(append(templatesHTML, templatesText...)),
	}

	http.Handle(
		"/html/{template}",
		logMiddleware(
			h.existMiddleware(
				liveReloadMiddleware(http.HandlerFunc(h.handleHTML)),
			),
		),
	)
	http.Handle(
		"/text/{template}",
		logMiddleware(
			h.existMiddleware(
				liveReloadMiddleware(http.HandlerFunc(h.handleTXT)),
			),
		),
	)

	http.Handle(
		"/ws/{template}",
		h.existMiddleware(http.HandlerFunc(wsH.wsHandle)),
	)

	srv := &http.Server{
		Addr:              "localhost:8080",
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)

	go func() {
		slog.Info(fmt.Sprintf("server started http://%s", srv.Addr))

		if err := srv.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
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

func (h *handler) handleHTML(w http.ResponseWriter, r *http.Request) {
	templ := r.PathValue("template")

	allFuncs := sprig.HtmlFuncMap()
	maps.Copy(allFuncs, text.FuncMap)

	t, err := template.New("").
		Funcs(allFuncs).
		ParseFiles(templ)
	if err != nil {
		slog.ErrorContext(r.Context(), "load template", slog.Any("error", err))
		http.Error(
			w,
			"Error while parse or load template file",
			http.StatusNotImplemented,
		)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = t.ExecuteTemplate(w, templ, h.data)
	if err != nil {
		slog.ErrorContext(
			r.Context(),
			"execute template",
			slog.Any("error", err),
		)
	}
}

const (
	textWrap = `<!doctype html>
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

	liveReloadScript = `
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
)

func (h *handler) handleTXT(w http.ResponseWriter, r *http.Request) {
	templ := r.PathValue("template")

	allFuncs := sprig.TxtFuncMap()
	maps.Copy(allFuncs, text.FuncMap)

	t, err := tmplTXT.New("").
		Funcs(allFuncs).
		ParseFiles(templ)
	if err != nil {
		slog.ErrorContext(r.Context(), "load template", slog.Any("error", err))
		http.Error(
			w,
			"Error while parse or load template file",
			http.StatusNotImplemented,
		)

		return
	}

	var buf bytes.Buffer

	err = t.ExecuteTemplate(&buf, templ, h.data)
	if err != nil {
		slog.ErrorContext(
			r.Context(),
			"execute template",
			slog.Any("error", err),
		)
	}

	dt := struct {
		TemplateName string
		Content      string
	}{
		TemplateName: templ,
		Content:      buf.String(),
	}

	tW, err := template.New("text_wrap").Parse(textWrap)
	if err != nil {
		slog.ErrorContext(r.Context(), "load template", slog.Any("error", err))
		http.Error(
			w,
			"Error while parse or load template file",
			http.StatusNotImplemented,
		)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = tW.ExecuteTemplate(w, "text_wrap", dt)
	if err != nil {
		slog.ErrorContext(
			r.Context(),
			"execute template",
			slog.Any("error", err),
		)
	}
}

func (h *handler) existMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templ := r.PathValue("template")
		if _, ok := h.templates[templ]; !ok {
			http.Error(w, "Not found template", http.StatusNotFound)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := &responseCapture{ResponseWriter: w}

		slog.InfoContext(
			r.Context(),
			"getting request",
			slog.Any("path", r.URL.String()),
		)

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

		if _, err := w.Write([]byte(buf)); err != nil {
			slog.ErrorContext(
				r.Context(),
				"write response",
				slog.Any("error", err),
			)
		}
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
				[]byte(
					fmt.Sprintf(
						liveReloadScript,
						url.PathEscape(r.PathValue("template")),
					)+"</body>",
				),
				1,
			)
		}

		if rc.statusCode != 0 {
			w.WriteHeader(rc.statusCode)
		}

		if _, err := w.Write(body); err != nil {
			slog.ErrorContext(
				r.Context(),
				"write live reload response",
				slog.Any("error", err),
			)
		}
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

			if err := conn.Close(); err != nil {
				slog.Error("close websocket", slog.Any("error", err))
			}
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
		if err := c.WriteMessage(
			websocket.TextMessage,
			[]byte("reload"),
		); err != nil {
			slog.Error("notify reload", slog.Any("error", err))
		}
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
	match, err := fs.Glob(f, "*.txt")
	if err != nil {
		log.Info("error check glob", slog.Any("error", err))

		return nil
	}

	if len(match) > 0 {
		log.Info(
			"loaded text templates",
			slog.Any("count", len(match)),
			slog.Any("templates", match),
		)
	}

	return match
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

func unmarshalFor[V any](data []byte) (V, error) {
	var v V

	err := json.Unmarshal(data, &v)

	return v, err
}
