package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"support_bot/internal/pkg/retry"
)

type Collector struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Collector {
	rt := retry.NewRoundTripper(http.DefaultTransport)
	client := &http.Client{Transport: rt, Timeout: cfg.Timeout}

	return &Collector{cfg: cfg, client: client}
}

func (c *Collector) Fetch(
	ctx context.Context,
	jql string,
	params map[string]string,
) ([]map[string]any, error) {
	u, err := url.Parse(c.cfg.JiraHost)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, "rest/api/2/search")

	q := u.Query()
	q.Set("jql", jql)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpl SearchRPL

	err = json.NewDecoder(resp.Body).Decode(&rpl)
	if err != nil {
		return nil, err
	}

	rpl.Flatten()

	return rpl.GetMap(), nil
}

func (c *Collector) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.cfg.AuthToken)
	req.Header.Set("Accept", "application/json")

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
