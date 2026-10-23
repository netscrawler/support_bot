package grafana

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"support_bot/internal/collector/prometheus"
	"support_bot/internal/pkg/retry"
)

const (
	defaultTimeout = 30 * time.Second
	defaultLimit   = 10000

	colAlertname = "alertname"
	paramFolder  = "folder"
	paramGroup   = "group"
	paramState   = "state"
	modeRules    = "rules"
	modeHistory  = "history"
)

// allowed: допустимые ключи params по режимам.
var allowed = map[string][]string{
	"active":    {paramFolder, paramGroup, paramState, colAlertname},
	modeRules:   {paramFolder, paramGroup, paramState, colAlertname},
	modeHistory: {"from", "to", "limit", colAlertname},
}

type source struct {
	baseURL string
	token   string
	timeout time.Duration
}

type Collector struct {
	sources map[string]source
	client  *http.Client
	log     *slog.Logger
}

func New(
	grafanas map[string]prometheus.GrafanaConfig,
	log *slog.Logger,
) *Collector {
	sources := make(map[string]source, len(grafanas))
	for name, g := range grafanas {
		sources[name] = source{
			baseURL: strings.TrimRight(g.URL, "/"),
			token:   g.Token.Value(),
			timeout: cmp.Or(g.Timeout, defaultTimeout),
		}
	}

	return &Collector{
		sources: sources,
		client: &http.Client{
			Transport: retry.NewRoundTripper(http.DefaultTransport),
		},
		log: log,
	}
}

type rulesResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		Groups []struct {
			Name  string `json:"name"`
			File  string `json:"file"`
			Rules []struct {
				Name           string            `json:"name"`
				State          string            `json:"state"`
				Health         string            `json:"health"`
				LastEvaluation time.Time         `json:"lastEvaluation"`
				Labels         map[string]string `json:"labels"`
				Alerts         []ruleAlert       `json:"alerts"`
			} `json:"rules"`
		} `json:"groups"`
	} `json:"data"`
}

type ruleAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
	Value       string            `json:"value"`
}

type annotation struct {
	Time      int64  `json:"time"`
	TimeEnd   int64  `json:"timeEnd"`
	AlertName string `json:"alertName"`
	PrevState string `json:"prevState"`
	NewState  string `json:"newState"`
	Text      string `json:"text"`
}

func (c *Collector) Fetch(
	ctx context.Context,
	target, mode string,
	params map[string]string,
) ([]map[string]any, error) {
	src, ok := c.sources[target]
	if !ok {
		return nil, fmt.Errorf("unknown grafana %q", target)
	}

	keys, ok := allowed[mode]
	if !ok {
		return nil, fmt.Errorf(
			"grafana %s: неизвестный режим %q (active, rules, history)",
			target,
			mode,
		)
	}

	for k := range params {
		if !slices.Contains(keys, k) {
			return nil, fmt.Errorf(
				"grafana %s: параметр %q недопустим для режима %s",
				target,
				k,
				mode,
			)
		}
	}

	path := "/api/prometheus/grafana/api/v1/rules"

	if mode == modeHistory {
		q, err := historyQuery(params)
		if err != nil {
			return nil, fmt.Errorf("grafana %s: %w", target, err)
		}

		path = "/api/annotations?" + q
	}

	c.log.DebugContext(
		ctx,
		"grafana fetch",
		slog.String("grafana", target),
		slog.String("mode", mode),
	)

	body, code, err := c.get(ctx, src, path)
	if err != nil {
		return nil, fmt.Errorf("grafana %s: %w", target, err)
	}

	var rows []map[string]any

	if mode == modeHistory {
		rows, err = historyRows(body, code)
	} else {
		rows, err = ruleRows(body, code, mode)
	}

	if err != nil {
		return nil, fmt.Errorf("grafana %s: %w", target, err)
	}

	return filter(rows, params, mode), nil
}

func (c *Collector) get(
	ctx context.Context,
	src source,
	path string,
) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(ctx, src.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		src.baseURL+path,
		http.NoBody,
	)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+src.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	return body, resp.StatusCode, err
}

func ok2xx(code int) bool { return code >= 200 && code < 300 }

func historyQuery(params map[string]string) (string, error) {
	limit := defaultLimit

	if s, ok := params["limit"]; ok {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return "", fmt.Errorf(
				"параметр limit: нужно положительное целое, получено %q",
				s,
			)
		}

		limit = n
	}

	v := url.Values{"type": {"alert"}, "limit": {strconv.Itoa(limit)}}

	for _, k := range []string{"from", "to"} {
		s, ok := params[k]
		if !ok {
			continue
		}

		if _, err := strconv.ParseInt(s, 10, 64); err != nil {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return "", fmt.Errorf(
					"параметр %s: ожидается epoch ms или RFC3339: %w",
					k,
					err,
				)
			}

			s = strconv.FormatInt(t.UnixMilli(), 10)
		}

		v.Set(k, s)
	}

	return v.Encode(), nil
}

func httpErr(code int, body []byte) error {
	return fmt.Errorf("HTTP %d: %.200s", code, body)
}

func historyRows(body []byte, code int) ([]map[string]any, error) {
	if !ok2xx(code) {
		return nil, httpErr(code, body)
	}

	var as []annotation

	if err := json.Unmarshal(body, &as); err != nil {
		return nil, fmt.Errorf("HTTP %d, ответ не JSON-массив: %w", code, err)
	}

	rows := make([]map[string]any, 0, len(as))
	for _, a := range as {
		rows = append(rows, map[string]any{
			"time":       fmtMS(a.Time),
			"time_end":   fmtMS(a.TimeEnd),
			colAlertname: alertName(a),
			"prev_state": a.PrevState,
			"new_state":  a.NewState,
			"text":       a.Text,
		})
	}

	return rows, nil
}

// alertName: unified alerting оставляет alertName пустым, имя правила — в text до « {».
func alertName(a annotation) string {
	if a.AlertName != "" {
		return a.AlertName
	}

	name, _, _ := strings.Cut(a.Text, " {")

	return name
}

func ruleRows(body []byte, code int, mode string) ([]map[string]any, error) {
	var r rulesResponse

	jerr := json.Unmarshal(body, &r)

	switch {
	case jerr == nil && r.Status != "":
		// Prometheus-формат разбираем и при 4xx/5xx: в нём есть текст ошибки.
		if r.Status != "success" {
			return nil, fmt.Errorf("%s: %s", r.ErrorType, r.Error)
		}

	case !ok2xx(code):
		return nil, httpErr(code, body)
	case jerr != nil:
		return nil, fmt.Errorf("HTTP %d, ответ не JSON: %w", code, jerr)
	default:
		return nil, fmt.Errorf(
			"HTTP %d, ответ не rules API: %.200s",
			code,
			body,
		)
	}

	var rows []map[string]any

	for _, g := range r.Data.Groups {
		for _, rule := range g.Rules {
			// Grafana кладёт в alerts и Normal-экземпляры («Normal (MissingSeries)» и т.п.) — активными их не считаем.
			rule.Alerts = slices.DeleteFunc(
				rule.Alerts,
				func(a ruleAlert) bool {
					return strings.HasPrefix(a.State, "Normal")
				},
			)

			if mode == modeRules {
				row := map[string]any{}
				copyLabels(row, rule.Labels)

				row["folder"] = g.File
				row["group"] = g.Name
				row["rule"] = rule.Name
				row["state"] = rule.State
				row["health"] = rule.Health
				row["last_evaluation"] = fmtTime(rule.LastEvaluation)
				row["active_instances"] = len(rule.Alerts)
				rows = append(rows, row)

				continue
			}

			for _, a := range rule.Alerts {
				row := map[string]any{}
				copyLabels(row, a.Labels)

				for k, v := range a.Annotations {
					row["annotation_"+k] = v
				}

				row[colAlertname] = rule.Name
				row["state"] = a.State
				row["active_at"] = fmtTime(a.ActiveAt)
				row["value"] = a.Value
				row["folder"] = g.File
				row["group"] = g.Name
				rows = append(rows, row)
			}
		}
	}

	return rows, nil
}

func copyLabels(row map[string]any, labels map[string]string) {
	for k, v := range labels {
		row[k] = v
	}
}

// fmtTime: нулевое время значит «ещё не было» -> nil.
func fmtTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}

	return t.UTC().Format(time.RFC3339)
}

func fmtMS(ms int64) any {
	if ms == 0 {
		return nil
	}

	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

// filter: точное совпадение; в rules alertname сравниваем с именем правила.
func filter(
	rows []map[string]any,
	params map[string]string,
	mode string,
) []map[string]any {
	nameCol := colAlertname
	if mode == modeRules {
		nameCol = "rule"
	}

	out := rows[:0]

next:
	for _, row := range rows {
		for _, k := range []string{paramFolder, paramGroup, paramState, colAlertname} {
			want, ok := params[k]
			if !ok {
				continue
			}

			col := k
			if k == colAlertname {
				col = nameCol
			}

			if row[col] != want {
				continue next
			}
		}

		out = append(out, row)
	}

	return out
}
