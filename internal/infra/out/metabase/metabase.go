package metabase

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"iter"
	"net/http"
	"time"

	"github.com/netscrawler/metabase-public-api"
)

type Metabase struct {
	client *metabase.Client
}

func New(baseURL string) *Metabase {
	rt := newRetraibleRoundTripper(http.DefaultTransport)
	client := http.Client{Transport: rt, Timeout: 5 * time.Minute}

	return &Metabase{client: metabase.NewClient(baseURL, &client)}
}

func (m *Metabase) GetDataMatrix(ctx context.Context, cardUUID string) ([][]string, error) {
	data, err := m.client.CardQuery(ctx, cardUUID, metabase.FormatCSV, nil)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(bytes.NewReader(data))

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (m *Metabase) GetDataMap(ctx context.Context, cardUUID string) ([]map[string]any, error) {
	data, err := m.client.CardQuery(ctx, cardUUID, metabase.FormatJSON, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]any

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *Metabase) GetDataIter(
	ctx context.Context,
	cardUUID string,
) (iter.Seq[map[string]any], error) {
	data, err := m.client.CardQuery(ctx, cardUUID, metabase.FormatJSON, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]any

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return func(yield func(map[string]any) bool) {
		for _, row := range result {
			if !yield(row) {
				return
			}
		}
	}, nil
}

type retraibleRoundTripper struct {
	t http.RoundTripper
}

func newRetraibleRoundTripper(tr http.RoundTripper) retraibleRoundTripper {
	return retraibleRoundTripper{t: tr}
}

func (x retraibleRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	retry := 3
	delay := 15 * time.Second

	for r := 0; ; r++ {
		resp, err := x.t.RoundTrip(req)
		if err == nil || r >= retry {
			return resp, err
		}

		select {
		case <-time.After(delay):
			delay = delay * 3
		case <-ctx.Done():
			return &http.Response{}, ctx.Err()
		}
	}
}
