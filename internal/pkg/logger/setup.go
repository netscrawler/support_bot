package logger

import (
	"log/slog"
	"os"
)

const (
	debug string = "debug"
	prod  string = "prod"
	test  string = "test"
)

func Setup(logCfg LogConfig) *slog.Logger {
	var (
		log  *slog.Logger
		opts *slog.HandlerOptions
	)

	//TODO: add multi writer support and logs management
	// logFile, err := os.OpenFile(
	// 	"log.log",
	// 	os.O_CREATE|os.O_APPEND|os.O_WRONLY,
	// 	0o644,
	// )
	// if err != nil {
	// 	panic(err)
	// }
	//
	// mw := io.MultiWriter(logFile, os.Stdout)

	switch logCfg.Level {
	case debug:
		opts = &slog.HandlerOptions{Level: slog.LevelDebug}
	case test:
		opts = &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true}
	case prod:
		opts = &slog.HandlerOptions{Level: slog.LevelInfo}
	default:
		opts = &slog.HandlerOptions{Level: slog.LevelInfo}
	}

	if logCfg.Level == test {
		log = slog.New(
			ContextHandler{
				Handler: slog.NewJSONHandler(
					os.Stdout,
					opts,
				),
			},
		)
	} else {
		log = slog.New(
			ContextHandler{
				Handler: slog.NewTextHandler(
					os.Stdout,
					opts,
				),
			},
		)
	}

	slog.SetDefault(log)

	return log
}
