package http

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var ErrAuthTokenRequired = fmt.Errorf("auth token is required")

type Config struct {
	Host              string        `env:"HTTP_HOST"          yaml:"host"          env-default:"127.0.0.1" comment:"HTTP server host"`
	Port              int           `env:"HTTP_PORT"          yaml:"port"          env-default:"8080"      comment:"HTTP server port"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT"  yaml:"read_timeout"  env-default:"5s"        comment:"HTTP server read timeout"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" yaml:"read_header_timeout" env-default:"5s"        comment:"HTTP server read header timeout"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT"       yaml:"write_timeout"       env-default:"6m"        comment:"HTTP server write timeout"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT"        yaml:"idle_timeout"        env-default:"120s"      comment:"HTTP server idle timeout"`

	MaxHeaderBytes      int `env:"HTTP_MAX_HEADER_BYTES" yaml:"max_header_bytes" env-default:"1048576" comment:"HTTP server max header bytes"`
	MaxHeaderValueCount int
	MaxBodyBytes        int64 `env:"HTTP_MAX_BODY_BYTES" yaml:"max_body_bytes" env-default:"10485760" comment:"HTTP server max body bytes"`

	// AuthToken has no default on purpose: the server refuses to start without it.
	AuthToken string `env:"AUTH_TOKEN" yaml:"auth_token" comment:"Bearer token required on every request"`
}

func (cfg *Config) Addr() string {
	return net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
}

func (cfg *Config) Validate() error {
	var errs []error

	if strings.TrimSpace(cfg.Host) == "" {
		errs = append(errs, errors.New("host is required"))
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		errs = append(errs, fmt.Errorf(
			"port must be between 1 and 65535, got %d",
			cfg.Port,
		))
	}

	if cfg.ReadTimeout <= 0 {
		errs = append(errs, fmt.Errorf(
			"read timeout must be positive, got %s",
			cfg.ReadTimeout,
		))
	}

	if cfg.WriteTimeout <= 0 {
		errs = append(errs, fmt.Errorf(
			"write timeout must be positive, got %s",
			cfg.WriteTimeout,
		))
	}

	if cfg.IdleTimeout <= 0 {
		errs = append(errs, fmt.Errorf(
			"idle timeout must be positive, got %s",
			cfg.IdleTimeout,
		))
	}

	if cfg.MaxBodyBytes <= 0 {
		errs = append(errs, fmt.Errorf(
			"max body bytes must be positive, got %d",
			cfg.MaxBodyBytes,
		))
	}

	if strings.TrimSpace(cfg.AuthToken) == "" {
		errs = append(errs, ErrAuthTokenRequired)
	}

	return errors.Join(errs...)
}
