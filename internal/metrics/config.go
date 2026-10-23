package metrics

type Config struct {
	Enabled bool   `env:"METRICS_ENABLED" yaml:"enabled" comment:"Enable /metrics and /healthz server"`
	Addr    string `env:"METRICS_ADDR"    yaml:"addr"    comment:"Metrics and healthz server address"  env-default:"127.0.0.1:9090"`
}
