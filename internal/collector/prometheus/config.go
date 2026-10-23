package prometheus

import (
	"cmp"
	"fmt"
	"net/http"
	"strings"
	"support_bot/internal/pkg/secret"
	"time"
)

const defaultTimeout = 30 * time.Second

type BasicAuth struct {
	User     string        `yaml:"user"     json:"user"     comment:"Пользователь"`
	Password secret.String `yaml:"password" json:"password" comment:"Пароль"`
}

type Config struct {
	URL           string        `yaml:"url"            json:"url"            comment:"URL Prometheus (прямой доступ)"`
	BearerToken   secret.String `yaml:"bearer_token"   json:"bearer_token"   comment:"Bearer-токен (необязательно, не вместе с basic_auth)"`
	BasicAuth     *BasicAuth    `yaml:"basic_auth"     json:"basic_auth"     comment:"Basic-авторизация (необязательно, не вместе с bearer_token)"`
	Grafana       string        `yaml:"grafana"        json:"grafana"        comment:"Имя записи из секции grafana (доступ через datasource proxy)"`
	DatasourceUID string        `yaml:"datasource_uid" json:"datasource_uid" comment:"UID datasource Prometheus в Grafana"`
	Timeout       time.Duration `yaml:"timeout"        json:"timeout"        comment:"Таймаут запроса (по умолчанию 30s, для grafana-записи — таймаут Grafana)"`
}

type GrafanaConfig struct {
	URL     string        `yaml:"url"     json:"url"     comment:"URL Grafana"`
	Token   secret.String `yaml:"token"   json:"token"   comment:"Токен service account (роль Viewer)"`
	Timeout time.Duration `yaml:"timeout" json:"timeout" comment:"Таймаут запроса (по умолчанию 30s)"`
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
		return source{}, fmt.Errorf("задайте либо url, либо grafana + datasource_uid")
	case c.URL == "" && !viaGrafana:
		return source{}, fmt.Errorf("не задан ни url, ни grafana + datasource_uid")
	case c.BearerToken != "" && c.BasicAuth != nil:
		return source{}, fmt.Errorf("bearer_token и basic_auth взаимоисключающие")
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
		return source{}, fmt.Errorf("для доступа через Grafana нужны и grafana, и datasource_uid")
	}

	g, ok := grafanas[c.Grafana]
	if !ok {
		return source{}, fmt.Errorf("grafana %q не найдена в секции grafana", c.Grafana)
	}

	return source{
		baseURL: strings.TrimRight(g.URL, "/") + "/api/datasources/proxy/uid/" + c.DatasourceUID,
		auth:    func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+g.Token.Value()) },
		timeout: cmp.Or(c.Timeout, g.Timeout, defaultTimeout),
	}, nil
}
