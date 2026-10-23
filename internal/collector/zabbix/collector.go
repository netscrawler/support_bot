package zabbix

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"support_bot/internal/pkg/funcs"
	"time"
)

const defaultTimeout = 30 * time.Second

var (
	placeholder = regexp.MustCompile(`\$\{(\w+)\}`)
	// Ровно один сегмент перед .get: Zabbix отбрасывает всё после второй точки (host.delete.get -> host.delete).
	readMethod = regexp.MustCompile(`^[a-z]+\.get$`)
)

type source struct {
	endpoint string
	token    string
	timeout  time.Duration
}

type Collector struct {
	sources map[string]source
	// Без retry.NewRoundTripper: он переотправляет уже прочитанное тело POST, повтор ушёл бы пустым.
	client *http.Client
	log    *slog.Logger
}

// New: валидацию делаем тут, т.к. env-default для значений map не работает.
func New(cfgs map[string]Config, log *slog.Logger) (*Collector, error) {
	sources := make(map[string]source, len(cfgs))

	for name, c := range cfgs {
		switch {
		case c.URL == "":
			return nil, fmt.Errorf("zabbix %q: не задан url", name)
		case c.Token == "":
			return nil, fmt.Errorf("zabbix %q: не задан token", name)
		}

		sources[name] = source{
			endpoint: strings.TrimRight(c.URL, "/") + "/api_jsonrpc.php",
			token:    c.Token.Value(),
			timeout:  cmp.Or(c.Timeout, defaultTimeout),
		}
	}

	return &Collector{sources: sources, client: &http.Client{}, log: log}, nil
}

type request struct {
	Method string `json:"method"`
	Params any    `json:"params"`
}

type response struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error"`
}

func (c *Collector) Fetch(
	ctx context.Context,
	target, query string,
	params map[string]string,
) ([]map[string]any, error) {
	src, ok := c.sources[target]
	if !ok {
		return nil, fmt.Errorf("unknown zabbix %q", target)
	}

	req, err := parseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("zabbix %s: %w", target, err)
	}

	if !readMethod.MatchString(req.Method) {
		return nil, fmt.Errorf(
			"zabbix %s: метод %s запрещён, разрешены только *.get",
			target,
			req.Method,
		)
	}

	p, err := substitute(req.Params, params)
	if err != nil {
		return nil, fmt.Errorf("zabbix %s: %w", target, err)
	}

	c.log.DebugContext(
		ctx,
		"zabbix fetch",
		slog.String("zabbix", target),
		slog.String("method", req.Method),
	)

	rows, err := c.call(ctx, src, req.Method, p)
	if err != nil {
		return nil, fmt.Errorf("zabbix %s: %w", target, err)
	}

	return rows, nil
}

func parseQuery(query string) (request, error) {
	var req request

	dec := json.NewDecoder(strings.NewReader(query))
	dec.DisallowUnknownFields()
	dec.UseNumber() // иначе time_from превратится в 1.7297e+09

	if err := dec.Decode(&req); err != nil {
		return req, fmt.Errorf("query: %w", err)
	}

	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, errors.New("query: лишние данные после JSON-объекта")
	}

	if req.Method == "" {
		return req, errors.New("query: не задан method")
	}

	if req.Params == nil {
		req.Params = map[string]any{}
	}

	return req, nil
}

// substitute: ${name} меняем только в строковых значениях, значение остаётся строкой JSON.
func substitute(v any, params map[string]string) (any, error) {
	switch x := v.(type) {
	case string:
		var missing string

		out := placeholder.ReplaceAllStringFunc(x, func(m string) string {
			name := placeholder.FindStringSubmatch(m)[1]

			val, ok := params[name]
			if !ok && missing == "" {
				missing = name
			}

			return val
		})
		if missing != "" {
			return nil, fmt.Errorf("в params нет значения для ${%s}", missing)
		}

		return out, nil
	case map[string]any:
		for k, e := range x {
			r, err := substitute(e, params)
			if err != nil {
				return nil, err
			}

			x[k] = r
		}
	case []any:
		for i, e := range x {
			r, err := substitute(e, params)
			if err != nil {
				return nil, err
			}

			x[i] = r
		}
	}

	return v, nil
}

func (c *Collector) call(
	ctx context.Context,
	src source,
	method string,
	params any,
) ([]map[string]any, error) {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": method, "params": params, "id": 1,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, src.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		src.endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json-rpc")
	req.Header.Set("Authorization", "Bearer "+src.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %.200s", resp.StatusCode, raw)
	}

	var r response
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("HTTP %d, ответ не JSON: %w", resp.StatusCode, err)
	}

	if r.Error != nil {
		return nil, fmt.Errorf("%s: %s", r.Error.Message, r.Error.Data)
	}

	// Без UseNumber: Lua-конвертер не знает json.Number.
	var result any
	if err := json.Unmarshal(r.Result, &result); err != nil {
		return nil, fmt.Errorf("HTTP %d, нет result: %w", resp.StatusCode, err)
	}

	return rows(result), nil
}

func rows(result any) []map[string]any {
	items, ok := result.([]any)
	if !ok {
		items = []any{result}
	}

	out := make([]map[string]any, 0, len(items))

	for _, it := range items {
		if obj, ok := it.(map[string]any); ok {
			out = append(out, funcs.Flatten(obj))
		} else {
			out = append(out, map[string]any{"result": it})
		}
	}

	return out
}
