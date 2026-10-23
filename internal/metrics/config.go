package metrics

type Config struct {
	Enabled bool   `comment:"Enable /metrics and /healthz server" env:"METRICS_ENABLED" yaml:"enabled"`
	Addr    string `comment:"Metrics and healthz server address"  env:"METRICS_ADDR"    env-default:"127.0.0.1:9090" yaml:"addr"`
}
