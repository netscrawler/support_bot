package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"

	"github.com/netscrawler/metabase-public-api"
)

type Metabase struct {
	client *metabase.Client
}

func NewMetabase(baseUrl string, client *http.Client) *Metabase {
	return &Metabase{client: metabase.NewClient(baseUrl, client)}
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

func (m *Metabase) GetDataMap(ctx context.Context, cardUUID string) (map[string]any, error) {
	data, err := m.client.CardQuery(ctx, cardUUID, metabase.FormatJSON, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]any

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result[0], nil
}
