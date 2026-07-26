package metabase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/netscrawler/metabase-public-api"
	"support_bot/internal/pkg/retry"
)

type Metabase struct {
	client *metabase.Client
}

func New(baseURL string) *Metabase {
	rt := retry.NewRoundTripper(http.DefaultTransport)
	client := http.Client{Transport: rt, Timeout: 5 * time.Minute}

	return &Metabase{client: metabase.NewClient(baseURL, &client)}
}

func (m *Metabase) Fetch(
	ctx context.Context,
	cardUUID string,
	params map[string]string,
) ([]map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("metabase query context : %w", err)
	}

	var filters []metabase.Filter

	for k, v := range params {
		filters = append(filters, metabase.NewCategoryFilter(k, v))
	}

	data, err := m.client.CardQuery(ctx, cardUUID, metabase.FormatJSON, filters)
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
