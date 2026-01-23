package metabase

import (
	"context"
	"encoding/json"
	"fmt"
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

func (m *Metabase) Fetch(ctx context.Context, cardUUID string) ([]map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metabase query context : %w", err)
	}

	data, err := m.client.CardQuery(ctx, cardUUID, metabase.FormatJSON, nil)
	if err != nil {
		return nil, fmt.Errorf("metabase card query : %w", err)
	}

	var result []map[string]any

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("metabase unmarshal query data : %w", err)
	}

	return result, nil
}
