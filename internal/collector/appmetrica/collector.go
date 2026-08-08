package appmetrica

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"support_bot/internal/pkg/retry"
)

type Collector struct {
	cfg    *Config
	client *http.Client

	log *slog.Logger
}

func NewCollector(cfg *Config, log *slog.Logger) *Collector {
	rt := retry.NewRoundTripper(http.DefaultTransport)
	client := &http.Client{Transport: rt, Timeout: cfg.Timeout}

	return &Collector{
		cfg:    cfg,
		client: client,
		log:    log,
	}
}

func (c *Collector) Fetch(
	ctx context.Context,
	target string,
	params map[string]string,
) ([]map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()

	for k, v := range params {
		q.Set(k, v)
	}

	req.URL.RawQuery = q.Encode()

	c.log.Debug("making request", "url", req.URL.String())

	raw, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer raw.Body.Close()

	c.log.Debug("parsing response", raw.Status)

	mediaType, _, err := mime.ParseMediaType(raw.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	switch mediaType {
	case "application/json":
		return c.processJSONResponse(raw)
	case "text/csv", "application/csv":
		return c.processCSVResponse(raw)
	default:
		return nil, fmt.Errorf("unexpected content type: %s", mediaType)
	}
}

func (c *Collector) GetApplications(ctx context.Context) ([]SupportedApplications, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.appmetrica.yandex.ru/management/v1/applications",
		http.NoBody,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var appResp GetMyApplicationResponse

	err = json.NewDecoder(resp.Body).Decode(&appResp)
	if err != nil {
		return nil, err
	}

	var apps []SupportedApplications

	for _, v := range appResp.Applications {
		apps = append(apps, SupportedApplications{
			AppName: v.Name,
			ID:      v.Id,
		})
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications found")
	}

	return apps, nil
}

func (c *Collector) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "OAuth "+c.cfg.OAuthToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return resp, nil
}

func (c *Collector) processJSONResponse(resp *http.Response) ([]map[string]any, error) {
	var report ReportDataResponse

	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(report.Data))

	for _, row := range report.Data {
		m := make(map[string]any)

		for i, name := range report.Query.Dimensions {
			if i < len(row.Dimensions) {
				m[name] = row.Dimensions[i].Name
			} else {
				m[name] = nil
			}
		}

		for i, name := range report.Query.Metrics {
			if i < len(row.Metrics) {
				m[name] = row.Metrics[i]
			} else {
				m[name] = nil
			}
		}

		result = append(result, m)
	}

	return result, nil
}

func (c *Collector) processCSVResponse(resp *http.Response) ([]map[string]any, error) {
	reader := bufio.NewReader(resp.Body)

	// Удаляем UTF-8 BOM, который часто добавляет AppMetrica
	bom, err := reader.Peek(3)
	if err == nil && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = reader.Discard(3)
	}

	csvReader := csv.NewReader(reader)

	// Иногда CSV от API может содержать большие поля
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true

	headers, err := csvReader.Read()
	if err == io.EOF {
		return []map[string]any{}, nil
	}

	if err != nil {
		return nil, err
	}

	// Чистим заголовки
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	result := make([]map[string]any, 0)

	for {
		record, err := csvReader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		row := make(map[string]any, len(headers))

		for i, header := range headers {
			if i < len(record) {
				row[header] = record[i]
			} else {
				row[header] = nil
			}
		}

		result = append(result, row)
	}

	return result, nil
}
