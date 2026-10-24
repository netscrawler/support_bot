package httpcoll

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
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"support_bot/internal/models"
	"support_bot/internal/pkg/retry"
)

const (
	defaultTimeout = 30 * time.Second
	errBodyLimit   = 512

	authBearer = "bearer"
	authBasic  = "basic"
	authHeader = "header"
)

// maxBody — переменная, чтобы тест мог её занизить.
var maxBody int64 = 50 << 20

// SecretGetter отдаёт расшифрованное значение секрета из БД и хост, к которому он привязан.
type SecretGetter interface {
	Get(ctx context.Context, name string) (value, host string, err error)
}

type Collector struct {
	sources map[string]Config
	secrets SecretGetter
	client  *http.Client
	log     *slog.Logger
}

// options — колонка options запроса.
type options struct {
	Headers    map[string]string `json:"headers"`
	Format     string            `json:"format"`
	Root       string            `json:"root"`
	Delimiter  string            `json:"delimiter"`
	Sheet      string            `json:"sheet"`
	Auth       string            `json:"auth"`
	AuthHeader string            `json:"auth_header"`
	Timeout    string            `json:"timeout"`
}

func validAuth(a string) bool {
	return a == "" || a == authBearer || a == authBasic || a == authHeader
}

// ParseHTTPURL принимает только абсолютный http(s)-URL с хостом.
func ParseHTTPURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
		u.Host == "" {
		return nil, errors.New("not an absolute http(s) url")
	}

	return u, nil
}

// New создаёт коллектор; secrets может быть nil (секреты из БД не поддерживаются).
func New(
	cfgs map[string]Config,
	secrets SecretGetter,
	log *slog.Logger,
) (*Collector, error) {
	for name, c := range cfgs {
		if _, err := ParseHTTPURL(c.BaseURL); err != nil {
			return nil, fmt.Errorf(
				"http source %q: invalid base_url: %w",
				name,
				err,
			)
		}

		if !validAuth(c.Auth) {
			return nil, fmt.Errorf(
				"http source %q: unknown auth %q",
				name,
				c.Auth,
			)
		}

		if c.Auth == authHeader && c.AuthHeader == "" {
			return nil, fmt.Errorf(
				"http source %q: auth_header is not set",
				name,
			)
		}
	}

	return &Collector{
		sources: cfgs,
		secrets: secrets,
		client: &http.Client{
			Transport:     retry.NewRoundTripper(http.DefaultTransport),
			CheckRedirect: sameHostRedirect,
		},
		log: log,
	}, nil
}

// sameHostRedirect не идёт на другой хост: net/http при этом снимает только
// Authorization и Cookie, а секрет в auth_header ушёл бы чужому серверу.
// Смена схемы тоже запрещена: понижение https->http выдало бы секрет открытым текстом.
// Ответ 3xx тогда возвращается как есть и даёт ошибку статуса.
func sameHostRedirect(req *http.Request, via []*http.Request) error {
	const maxRedirects = 10

	if len(via) >= maxRedirects {
		return errors.New("stopped after 10 redirects")
	}

	if req.URL.Host != via[0].URL.Host || req.URL.Scheme != via[0].URL.Scheme {
		return http.ErrUseLastResponse
	}

	return nil
}

func parseOptions(raw json.RawMessage) (options, time.Duration, error) {
	var o options

	if len(bytes.TrimSpace(raw)) > 0 {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&o); err != nil {
			return o, 0, fmt.Errorf("invalid options: %w", err)
		}
	}

	switch o.Format {
	case "", formatJSON, formatNDJSON, formatCSV,
		formatXML, formatXLSX, formatYAML, formatText:
	default:
		return o, 0, fmt.Errorf("invalid options: unknown format %q", o.Format)
	}

	if !validAuth(o.Auth) {
		return o, 0, fmt.Errorf("invalid options: unknown auth %q", o.Auth)
	}

	if o.Delimiter != "" && utf8.RuneCountInString(o.Delimiter) != 1 {
		return o, 0, fmt.Errorf(
			"invalid options: delimiter %q must be one character",
			o.Delimiter,
		)
	}

	if o.Timeout == "" {
		return o, 0, nil
	}

	timeout, err := time.ParseDuration(o.Timeout)
	if err != nil || timeout <= 0 {
		return o, 0, fmt.Errorf("invalid options: bad timeout %q", o.Timeout)
	}

	return o, timeout, nil
}

// ValidateOptions проверяет options карточки без запроса (для валидатора отчётов).
// auth_header не обязателен: имя заголовка может прийти из конфига источника.
func ValidateOptions(raw json.RawMessage) error {
	_, _, err := parseOptions(raw)

	return err
}

func (c *Collector) Fetch(
	ctx context.Context,
	target, query string,
	params map[string]string,
) ([]map[string]any, error) {
	return c.FetchCard(
		ctx,
		models.Card{CardUUID: target, Query: query, Params: params},
	)
}

func (c *Collector) FetchCard(
	ctx context.Context,
	card models.Card,
) ([]map[string]any, error) {
	var secrets []string // значения для вычистки из ошибок

	rows, err := c.fetch(ctx, card, &secrets)
	if err == nil {
		return rows, nil
	}

	msg := err.Error()
	for _, s := range secrets {
		if s != "" {
			msg = strings.ReplaceAll(msg, s, "***")
		}
	}

	// Без секрета в тексте отдаём исходную ошибку: metrics.Reason смотрит цепочку.
	if msg == err.Error() {
		return nil, err
	}

	return nil, errors.New(msg)
}

func (c *Collector) fetch(
	ctx context.Context,
	card models.Card,
	secrets *[]string,
) ([]map[string]any, error) {
	opts, optTimeout, err := parseOptions(card.Options)
	if err != nil {
		return nil, err
	}

	cfg, named := c.sources[card.CardUUID]

	var u *url.URL

	if named {
		base := strings.TrimRight(cfg.BaseURL, "/")

		u, err = url.Parse(base + "/" + strings.TrimLeft(card.Query, "/"))
		if err != nil {
			return nil, fmt.Errorf(
				"source %q: invalid url: %w",
				card.CardUUID,
				err,
			)
		}
	} else if u, err = ParseHTTPURL(card.CardUUID); err != nil {
		return nil, fmt.Errorf(
			"card_uuid %q is neither a configured source nor %w",
			card.CardUUID,
			err,
		)
	}

	q := u.Query()
	for k, v := range card.Params {
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()

	// В ошибки идёт только scheme+host+path: в query может быть ключ API.
	safeURL := u.Scheme + "://" + u.Host + u.Path

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		u.String(),
		http.NoBody,
	)
	if err != nil {
		return nil, fmt.Errorf("build request %s: %w", safeURL, err)
	}

	for _, h := range []map[string]string{cfg.Headers, opts.Headers} {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	if authErr := c.authorize(
		ctx,
		req,
		u,
		cfg,
		opts,
		card.Secret,
		secrets,
	); authErr != nil {
		return nil, authErr
	}

	timeout := cmp.Or(optTimeout, cfg.Timeout, defaultTimeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		if ue, ok := errors.AsType[*url.Error](err); ok {
			err = ue.Err
		}

		return nil, fmt.Errorf("http get %s: %w", safeURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("read response %s: %w", safeURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf(
			"http %s: status %d: %s",
			safeURL, resp.StatusCode, body[:min(len(body), errBodyLimit)],
		)
	}

	if int64(len(body)) > maxBody {
		return nil, errors.New("response body exceeds 50 MB")
	}

	return decode(
		opts.Format,
		resp.Header.Get("Content-Type"),
		opts.Root,
		opts.Delimiter,
		opts.Sheet,
		body,
	)
}

// authorize выставляет авторизацию: секрет карточки перекрывает конфиг источника.
func (c *Collector) authorize(
	ctx context.Context,
	req *http.Request,
	u *url.URL,
	cfg Config,
	opts options,
	secretName string,
	secrets *[]string,
) error {
	mode := cmp.Or(opts.Auth, cfg.Auth)
	hdr := cmp.Or(opts.AuthHeader, cfg.AuthHeader)
	value := cfg.Token.Value()
	user, pass := cfg.User, cfg.Password.Value()

	*secrets = append(*secrets, value, pass)

	if secretName != "" {
		if c.secrets == nil {
			return fmt.Errorf(
				"secret %q: secrets are not supported",
				secretName,
			)
		}

		v, host, err := c.secrets.Get(ctx, secretName)
		if err != nil {
			return fmt.Errorf("secret %q: %w", secretName, err)
		}

		// Секрет уходит только на свой хост: "host:port" сравнивается с портом,
		// "host" (в том числе IPv6 без скобок) — с любым портом.
		if !strings.EqualFold(host, u.Host) &&
			!strings.EqualFold(host, u.Hostname()) {
			return fmt.Errorf(
				"secret %q is bound to host %q, request host %q",
				secretName, host, u.Host,
			)
		}

		*secrets = append(*secrets, v)
		mode, value = cmp.Or(opts.Auth, authBearer), v
	}

	switch mode {
	case authBearer:
		req.Header.Set("Authorization", "Bearer "+value)
	case authBasic:
		if secretName != "" { // секрет basic — "user:password"
			var ok bool

			if user, pass, ok = strings.Cut(value, ":"); !ok {
				return fmt.Errorf(
					"secret %q: basic auth value must be user:password",
					secretName,
				)
			}

			*secrets = append(*secrets, pass)
		}

		req.SetBasicAuth(user, pass)

	case authHeader:
		if hdr == "" {
			return errors.New("auth_header is not set")
		}

		req.Header.Set(hdr, value)
	}

	return nil
}
