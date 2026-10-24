package postgres

import (
	"net"
	"net/url"
	"strconv"
	"time"

	"support_bot/internal/pkg/secret"
)

const (
	sslDisable    = "disable"
	sslRequire    = "require"
	sslVerifyCa   = "verify-ca"
	sslVerifyFull = "verify-full"
	sslAllow      = "allow"
	sslPrefer     = "prefer"
)

type Config struct {
	Port     int           `comment:"Порт для подключения к базе"                                          env:"DATABASE_PORT"     env-default:"5432"      yaml:"port"`
	Host     string        `comment:"Адрес базы данных"                                                    env:"DATABASE_HOST"     env-default:"localhost" yaml:"host"`
	User     string        `Comment:"Пользователь"                                                         env:"DATABASE_USER"     env-default:"user"      yaml:"user"`
	Password secret.String `comment:"Пароль для подключения"                                               env:"DATABASE_PASSWORD" env-default:"password"  yaml:"password"`
	Name     string        `comment:"Имя базы данных"                                                      env:"DATABASE_NAME"     env-default:"postgres"  yaml:"name"`
	SSL      string        `comment:"SSL режим: (disable|require|verify-ca|verify-full|allow|prefer)"      env:"DATABASE_SSL_MODE" env-default:"false"     yaml:"sslmode"`

	MaxConns        int           `comment:"Максимально количество соединений"                                                     env:"DATABASE_MAX_CONNS"          env-default:"10"  yaml:"max_conns"`
	MaxIdleConns    int           `comment:"Максимальное количество ожидающий соединений"                                          env:"DATABASE_MAX_IDLE_CONNS"     env-default:"2"   yaml:"max_idle_conns"`
	MaxConnLifeTime time.Duration `comment:"Максимальное время жизни одного соединения"                                            env:"DATABASE_MAX_CONN_LIFE_TIME" env-default:"30m" yaml:"max_conn_life_time"`
	MaxConnIdleTime time.Duration `comment:"Максимальное время ожидания соединения"                                                env:"DATABASE_MAX_CONN_IDLE_TIME" env-default:"5m"  yaml:"max_conn_idle_time"`

	DatabaseConnect time.Duration `comment:"Таймаут на подключения к базе данных" env:"DATABASE_CONNECT_TIMEOUT" env-default:"30s" yaml:"database_connect"`

	DSN secret.String `yaml:"-"`
}

func (pc *Config) getDSN() secret.String {
	mode := sslDisable
	modes := map[string]struct{}{
		sslDisable:    {},
		sslRequire:    {},
		sslVerifyCa:   {},
		sslVerifyFull: {},
		sslAllow:      {},
		sslPrefer:     {},
	}

	if _, ok := modes[pc.SSL]; ok {
		mode = pc.SSL
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(pc.User, pc.Password.Value()),
		Host:   net.JoinHostPort(pc.Host, strconv.Itoa(pc.Port)),
		Path:   pc.Name,
	}

	q := dsn.Query()
	q.Set("sslmode", mode)
	q.Set("connect_timeout", strconv.Itoa(int(pc.DatabaseConnect.Seconds())))

	dsn.RawQuery = q.Encode()

	pc.DSN = secret.String(dsn.String())
	// в DSN пароль URL-кодирован и может не совпасть с зарегистрированным Password
	secret.Register(pc.DSN)

	return pc.DSN
}
