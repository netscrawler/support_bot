package logger

type LogConfig struct {
	Level  string   `env:"LOG_LEVEL"  yaml:"level"`
	File   string   `env:"LOG_FILE"   yaml:"file"`
	Output []string `env:"LOG_OUTPUT" yaml:"output"`
}
