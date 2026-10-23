package http

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host              string        `comment:"HTTP server host"                env:"HTTP_HOST"                env-default:"0.0.0.0" yaml:"host"`
	Port              int           `comment:"HTTP server port"                env:"HTTP_PORT"                env-default:"8080"    yaml:"port"`
	ReadTimeout       time.Duration `comment:"HTTP server read timeout"        env:"HTTP_READ_TIMEOUT"        env-default:"5s"      yaml:"read_timeout"`
	ReadHeaderTimeout time.Duration `comment:"HTTP server read header timeout" env:"HTTP_READ_HEADER_TIMEOUT" env-default:"5s"      yaml:"read_header_timeout"`
	WriteTimeout      time.Duration `comment:"HTTP server write timeout"       env:"HTTP_WRITE_TIMEOUT"       env-default:"6m"      yaml:"write_timeout"`
	IdleTimeout       time.Duration `comment:"HTTP server idle timeout"        env:"HTTP_IDLE_TIMEOUT"        env-default:"120s"    yaml:"idle_timeout"`

	MaxHeaderBytes      int   `comment:"HTTP server max header bytes"       env:"HTTP_MAX_HEADER_BYTES"  env-default:"1048576"  yaml:"max_header_bytes"`
	MaxHeaderValueCount int   `comment:"HTTP server max header value count" env:"HTTP_MAX_HEADER_VALUES" env-default:"40"       yaml:"max_header_value_count"`
	MaxBodyBytes        int64 `comment:"HTTP server max body bytes"         env:"HTTP_MAX_BODY_BYTES"    env-default:"10485760" yaml:"max_body_bytes"`
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

	return errors.Join(errs...)
}
