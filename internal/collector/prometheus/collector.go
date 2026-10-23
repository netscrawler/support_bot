package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"support_bot/internal/pkg/retry"
	"time"
)

type Collector struct {
	sources map[string]source
	client  *http.Client
	log     *slog.Logger
}

func New(
	cfgs map[string]Config,
	grafanas map[string]GrafanaConfig,
	log *slog.Logger,
) (*Collector, error) {
	sources, err := buildSources(cfgs, grafanas)
	if err != nil {
		return nil, err
	}

	return &Collector{
		sources: sources,
		client:  &http.Client{Transport: retry.NewRoundTripper(http.DefaultTransport)},
		log:     log,
	}, nil
}

type apiResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		ResultType string          `json:"resultType"`
		Result     json.RawMessage `json:"result"`
	} `json:"data"`
	Warnings []string `json:"warnings"`
}

type sample struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
	Values [][]any           `json:"values"`
}

func (c *Collector) Fetch(
	ctx context.Context,
	target, query string,
	params map[string]string,
) ([]map[string]any, error) {
	src, ok := c.sources[target]
	if !ok {
		return nil, fmt.Errorf("unknown prometheus source %q", target)
	}

	path, vals, err := buildRequest(query, params)
	if err != nil {
		return nil, fmt.Errorf("prometheus %s: %w", target, err)
	}

	ctx, cancel := context.WithTimeout(ctx, src.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, src.baseURL+path+"?"+vals.Encode(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("prometheus %s: %w", target, err)
	}

	req.Header.Set("Accept", "application/json")

	if src.auth != nil {
		src.auth(req)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus %s: %w", target, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("prometheus %s: %w", target, err)
	}

	// Декодируем при любом статусе: Prometheus отдаёт JSON-ошибки с 400/422/503.
	var r apiResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf(
			"prometheus %s: HTTP %d, ответ не JSON: %w",
			target,
			resp.StatusCode,
			err,
		)
	}

	// JSON без status — не Prometheus API (например, 401 от Grafana: {"message": ...}).
	if r.Status == "" {
		return nil, fmt.Errorf("prometheus %s: HTTP %d: %.200s", target, resp.StatusCode, body)
	}

	if r.Status != "success" {
		return nil, fmt.Errorf("prometheus %s: %s: %s", target, r.ErrorType, r.Error)
	}

	for _, w := range r.Warnings {
		c.log.WarnContext(ctx, "prometheus warning",
			slog.String("source", target), slog.String("warning", w))
	}

	rows, err := toRows(r.Data.ResultType, r.Data.Result)
	if err != nil {
		return nil, fmt.Errorf("prometheus %s: %w", target, err)
	}

	return rows, nil
}

func toRows(resultType string, raw json.RawMessage) ([]map[string]any, error) {
	if resultType != "vector" && resultType != "matrix" {
		return nil, fmt.Errorf("неподдерживаемый тип результата %q", resultType)
	}

	var series []sample
	if err := json.Unmarshal(raw, &series); err != nil {
		return nil, fmt.Errorf("разбор результата: %w", err)
	}

	rows := make([]map[string]any, 0, len(series))

	for _, s := range series {
		points := s.Values
		if s.Value != nil {
			points = [][]any{s.Value}
		}

		// Сэмпл native histogram приходит без value/values: одна строка с value=nil.
		if len(points) == 0 {
			points = [][]any{nil}
		}

		for _, p := range points {
			row := make(map[string]any, len(s.Metric)+2)

			for k, v := range s.Metric {
				if k == "__name__" {
					k = "metric"
				}

				row[k] = v
			}

			row["value"] = nil

			if len(p) == 2 {
				if resultType == "matrix" {
					row["timestamp"] = formatTS(p[0])
				}

				row["value"] = parseValue(p[1])
			}

			rows = append(rows, row)
		}
	}

	return rows, nil
}

func formatTS(v any) any {
	f, ok := v.(float64)
	if !ok {
		return nil
	}

	sec, frac := math.Modf(f)

	return time.Unix(int64(sec), int64(frac*1e9)).UTC().Format(time.RFC3339)
}

func parseValue(v any) any {
	s, ok := v.(string)
	if !ok {
		return nil
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}

	return f
}
