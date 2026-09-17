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
	Host              string        `env:"HTTP_HOST"                yaml:"host"                env-default:"0.0.0.0" comment:"HTTP server host"`
	Port              int           `env:"HTTP_PORT"                yaml:"port"                env-default:"8080"    comment:"HTTP server port"`
	ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT"        yaml:"read_timeout"        env-default:"5s"      comment:"HTTP server read timeout"`
	ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" yaml:"read_header_timeout" env-default:"5s"      comment:"HTTP server read header timeout"`
	WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT"       yaml:"write_timeout"       env-default:"6m"      comment:"HTTP server write timeout"`
	IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT"        yaml:"idle_timeout"        env-default:"120s"    comment:"HTTP server idle timeout"`

	MaxHeaderBytes      int   `env:"HTTP_MAX_HEADER_BYTES"  yaml:"max_header_bytes"       env-default:"1048576"  comment:"HTTP server max header bytes"`
	MaxHeaderValueCount int   `env:"HTTP_MAX_HEADER_VALUES" yaml:"max_header_value_count" env-default:"40"       comment:"HTTP server max header value count"`
	MaxBodyBytes        int64 `env:"HTTP_MAX_BODY_BYTES"    yaml:"max_body_bytes"         env-default:"10485760" comment:"HTTP server max body bytes"`
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
