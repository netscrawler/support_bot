package logger

import (
	"errors"
	"fmt"
)

const (
	logOutStdout = "stdout"
	logOutStdin  = "stdin"
	logOutStderr = "stderr"
	logOutFile   = "file"
)

const (
	debug string = "debug"
	prod  string = "prod"
	test  string = "test"
)

const (
	fieldTime   = "time"
	fieldLevel  = "level"
	fieldMSG    = "msg"
	fieldSource = "source"
)

type LogConfig struct {
	Level   string   `env:"LOG_LEVEL"   yaml:"level"   env-default:"prod"   comment:"Профиль логирования (debug - уровень DEBUG|prod - Уровень INFO|test - Уровень DEBUG + add source)"`
	File    string   `env:"LOG_FILE"    yaml:"file"                         comment:"Путь к файлу куда писать логи(работает только если в output есть file)"`
	Output  []string `env:"LOG_OUTPUT"  yaml:"output"  env-default:"stdout" comment:"Вывод логов: (stdout|stdin|stderr|file) поддерживает несколько"`
	Format  string   `env:"LOG_FORMAT"  yaml:"format"  env-default:"text"   comment:"Формат в котором пишутся логи: (json|text)"`
	Exclude []string `env:"LOG_EXCLUDE" yaml:"exclude"                      comment:"Список стандартных полей для исключения (time, level, msg, source)"`
}

func (l *LogConfig) Default() {
	if l.Level == "" {
		l.Level = debug
	}

	if l.Format == "" {
		l.Format = "text"
	}

	if len(l.Output) == 0 {
		l.Output = append(l.Output, logOutStdout)
	}
}

func (l *LogConfig) Validate() error {
	levelErr := validateLevel(l.Level)

	formatErr := validateFormat(l.Format)

	var outErr error

	for _, o := range l.Output {
		valid := validateOut(o)
		if !valid {
			outErr = errors.Join(outErr, fmt.Errorf("invalid log format: %s", o))
		}
	}

	for _, o := range l.Exclude {
		err := validateExcluded(o)
		if err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	if outErr != nil {
		outErr = fmt.Errorf("%w available out:(file, stdout, stdin, stderr)", outErr)
	}

	return errors.Join(levelErr, formatErr, outErr)
}

func validateOut(out string) bool {
	switch out {
	case logOutFile, logOutStderr, logOutStdin, logOutStdout:
		return true
	default:
		return false
	}
}

func validateFormat(format string) error {
	switch format {
	case "json", "text":
		return nil
	default:
		return fmt.Errorf(
			"invalid log format: %s, available formats:(json, text)",
			format,
		)
	}
}

func validateLevel(level string) error {
	switch level {
	case prod, debug, test:
		return nil
	default:
		return fmt.Errorf(
			"invalid log level: %s, available formats:(debug, test, prod)",
			level,
		)
	}
}

func validateExcluded(field string) error {
	switch field {
	case fieldTime, fieldSource, fieldLevel, fieldMSG:
		return nil
	default:
		return fmt.Errorf(
			"invalid excluded field: %s, available fields:(time, level, msg, source)",
			field,
		)
	}
}
