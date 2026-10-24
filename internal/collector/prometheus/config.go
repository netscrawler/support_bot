package prometheus

import (
	"cmp"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"support_bot/internal/pkg/secret"
)

const (
	defaultTimeout = 30 * time.Second
)

type BasicAuth struct {
	User     string        `comment:"Пользователь"             json:"user"     yaml:"user"`
	Password secret.String `comment:"Пароль"                   json:"password" yaml:"password"`
}

type Config struct {
	URL           string        `comment:"URL Prometheus (прямой доступ)"                                                                                      json:"url"            yaml:"url"`
	BearerToken   secret.String `comment:"Bearer-токен (необязательно, не вместе с basic_auth)"                                                                json:"bearer_token"   yaml:"bearer_token"`
	BasicAuth     *BasicAuth    `comment:"Basic-авторизация (необязательно, не вместе с bearer_token)"                                                         json:"basic_auth"     yaml:"basic_auth"`
	Grafana       string        `comment:"Имя записи из секции grafana (доступ через datasource proxy)"                                                        json:"grafana"        yaml:"grafana"`
	DatasourceUID string        `comment:"UID datasource Prometheus в Grafana"                                                                                 json:"datasource_uid" yaml:"datasource_uid"`
	Timeout       time.Duration `comment:"Таймаут запроса (по умолчанию 30s, для grafana-записи — таймаут Grafana)"                                            json:"timeout"        yaml:"timeout"`
}

type GrafanaConfig struct {
	URL     string        `comment:"URL Grafana"                                                 json:"url"     yaml:"url"`
	Token   secret.String `comment:"Токен service account (роль Viewer)"                         json:"token"   yaml:"token"`
	Timeout time.Duration `comment:"Таймаут запроса (по умолчанию 30s)"                          json:"timeout" yaml:"timeout"`
}

type source struct {
	baseURL string
	auth    func(*http.Request)
	timeout time.Duration
}

// buildSources: дефолты и валидацию делаем тут, т.к. env-default для значений map не работает.
func buildSources(
	cfgs map[string]Config,
	grafanas map[string]GrafanaConfig,
) (map[string]source, error) {
	out := make(map[string]source, len(cfgs))

	for name, c := range cfgs {
		s, err := buildSource(c, grafanas)
		if err != nil {
			return nil, fmt.Errorf("prometheus %q: %w", name, err)
		}

		out[name] = s
	}

	return out, nil
}

func buildSource(c Config, grafanas map[string]GrafanaConfig) (source, error) {
	viaGrafana := c.Grafana != "" || c.DatasourceUID != ""

	switch {
	case c.URL != "" && viaGrafana:
		return source{}, errors.New(
			"set either url or grafana + datasource_uid",
		)

	case c.URL == "" && !viaGrafana:
		return source{}, errors.New(
			"neither url nor grafana + datasource_uid is set",
		)

	case c.BearerToken != "" && c.BasicAuth != nil:
		return source{}, errors.New(
			"bearer_token and basic_auth are mutually exclusive",
		)
	}

	if !viaGrafana {
		s := source{
			baseURL: strings.TrimRight(c.URL, "/"),
			timeout: cmp.Or(c.Timeout, defaultTimeout),
		}

		switch {
		case c.BearerToken != "":
			s.auth = func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+c.BearerToken.Value()) }
		case c.BasicAuth != nil:
			s.auth = func(r *http.Request) { r.SetBasicAuth(c.BasicAuth.User, c.BasicAuth.Password.Value()) }
		}

		return s, nil
	}

	if c.Grafana == "" || c.DatasourceUID == "" {
		return source{}, errors.New(
			"access via Grafana requires both grafana and datasource_uid",
		)
	}

	g, ok := grafanas[c.Grafana]
	if !ok {
		return source{}, fmt.Errorf(
			"grafana %q not found in grafana section",
			c.Grafana,
		)
	}

	return source{
		baseURL: strings.TrimRight(
			g.URL,
			"/",
		) + "/api/datasources/proxy/uid/" + c.DatasourceUID,
		auth:    func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+g.Token.Value()) },
		timeout: cmp.Or(c.Timeout, g.Timeout, defaultTimeout),
	}, nil
}
