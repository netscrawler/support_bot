package logger

import (
	"io"
	"log/slog"
	"os"
)

func Setup(logCfg LogConfig) (*slog.Logger, error) {
	mw, err := getWriters(logCfg)
	if err != nil {
		return nil, err
	}

	opts := getOpts(logCfg.Level)

	log := getLogger(logCfg.Format, mw, opts)

	slog.SetDefault(log)

	return log, nil
}

func getLogger(format string, writer io.Writer, opts *slog.HandlerOptions) *slog.Logger {
	if format == "json" {
		return slog.New(
			ContextHandler{Handler: slog.NewJSONHandler(writer, opts)},
		)
	}

	return slog.New(
		ContextHandler{Handler: slog.NewTextHandler(writer, opts)},
	)
}

func getOpts(level string) *slog.HandlerOptions {
	var opts *slog.HandlerOptions

	switch level {
	case debug:
		opts = &slog.HandlerOptions{Level: slog.LevelDebug}
	case test:
		opts = &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true}
	case prod:
		opts = &slog.HandlerOptions{Level: slog.LevelInfo}
	default:
		opts = &slog.HandlerOptions{Level: slog.LevelInfo}
	}

	return opts
}

func getWriters(logCfg LogConfig) (io.Writer, error) {
	writers := []io.Writer{}

	var err error

	for _, form := range logCfg.Output {
		switch form {
		case LogOutStdout:
			writers = append(writers, os.Stdout)
		case LogOutStderr:
			writers = append(writers, os.Stderr)
		case LogOutStdin:
			writers = append(writers, os.Stdin)
		case LogOutFile:
			logFile, cErr := os.OpenFile(
				logCfg.File,
				os.O_CREATE|os.O_APPEND|os.O_WRONLY,
				0o600,
			)
			err = cErr

			writers = append(writers, logFile)
		}
	}

	return io.MultiWriter(writers...), err
}
