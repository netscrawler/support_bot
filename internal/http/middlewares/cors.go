package middlewares

import (
	"net/http"
	"strconv"
	"strings"
)

type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins" env-default:"*"`
	AllowedMethods   []string `yaml:"allowed_methods" env-default:"GET,POST,PUT,PATCH,DELETE,OPTIONS"`
	AllowedHeaders   []string `yaml:"allowed_headers" env-default:"Content-Type,Accept,Authorization,X-Trace-ID"`
	ExposedHeaders   []string `env-default:"X-Trace-ID"`
	AllowCredentials bool     `yaml:"allow_credentials" env-default:"true"`
	MaxAge           int      `yaml:"max_age" env-default:"600"`
}

type CORSMiddleware struct {
	enabled bool
	cfg     CORSConfig
	allowed map[string]struct{}
}

func NewCORSMiddleware(cfg CORSConfig) *CORSMiddleware {
	allowed := make(map[string]struct{})
	for _, o := range cfg.AllowedOrigins {
		allowed[o] = struct{}{}
	}

	return &CORSMiddleware{
		enabled: cfg.Enabled,
		cfg:     cfg,
		allowed: allowed,
	}
}

func (c *CORSMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !c.enabled {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Проверка origin
		if _, ok := c.allowed[origin]; ok || c.isWildcard() {

			if c.cfg.AllowCredentials && c.isWildcard() {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			} else if c.isWildcard() {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}

			if c.cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods",
				strings.Join(c.cfg.AllowedMethods, ","))

			w.Header().Set("Access-Control-Allow-Headers",
				strings.Join(c.cfg.AllowedHeaders, ","))

			w.Header().Set("Access-Control-Expose-Headers",
				strings.Join(c.cfg.ExposedHeaders, ","))

			w.Header().Set("Access-Control-Max-Age",
				strconv.Itoa(c.cfg.MaxAge))
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *CORSMiddleware) isWildcard() bool {
	return len(c.cfg.AllowedOrigins) == 1 &&
		c.cfg.AllowedOrigins[0] == "*"
}
