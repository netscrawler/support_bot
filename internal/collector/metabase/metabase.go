package metabase

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
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

func (m *Metabase) FetchMatrix(ctx context.Context, cardUUID string) ([][]string, error) {
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

func (m *Metabase) Fetch(ctx context.Context, cardUUID string) ([]map[string]any, error) {
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
