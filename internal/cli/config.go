package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"support_bot/internal/config"
	"support_bot/internal/pkg"

	"gopkg.in/yaml.v3"
)

func Config() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s config <command> [args]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands: validate, update, create\n")
		return
	}

	var err error
	switch os.Args[2] {
	case "validate":
		err = configValidate(os.Args[3:])
	case "update":
		err = configUpdate(os.Args[3:])
	case "create":
		err = configCreate(os.Args[3:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown config command: %s\n", os.Args[2])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func configValidate(args []string) error {
	fs := flag.NewFlagSet("config validate", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "Путь к файлу конфигурации")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *cfgPath == "" {
		return fmt.Errorf("config path is required")
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Config validation successful\n")
	return nil
}

func configUpdate(args []string) error {
	fs := flag.NewFlagSet("config update", flag.ContinueOnError)

	cfgPath := fs.String("config", "", "Путь к файлу конфигурации")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *cfgPath == "" {
		return fmt.Errorf("config path is required")
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var cfgStr string

	ext := filepath.Ext(*cfgPath)
	cfgStr, err = marshalConfig(cfg, ext)
	if err != nil {
		return err
	}

	err = os.WriteFile(*cfgPath, []byte(cfgStr), 0o644)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func configCreate(args []string) error {
	fs := flag.NewFlagSet("config create", flag.ContinueOnError)

	out := fs.String(
		"out",
		"",
		"Путь к файлу для сохранения конфигурации (по умолчанию вывод в stdout)",
	)
	format := fs.String("format", "yaml", "Формат конфигурации (yaml, env)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var cfg string
	var err error

	cfg, err = marshalConfig(config.Default(), *format)
	if err != nil {
		return err
	}

	var outW io.Writer

	if *out == "" {
		outW = os.Stdout
	} else {
		f, err := os.OpenFile(*out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("open output file: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()
		outW = f
	}

	_, err = outW.Write([]byte(cfg))
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func marshalConfig(cfg *config.Config, format string) (string, error) {
	switch format {
	case ".yaml", ".yml", "yaml", "yml":
		node, err := pkg.StructToYAMLNode(cfg)
		if err != nil {
			return "", fmt.Errorf("create config node: %w", err)
		}

		mCfg, err := yaml.Marshal(node)
		if err != nil {
			return "", fmt.Errorf("marshal config: %w", err)
		}
		return string(mCfg), nil

	case ".env", "env":
		defEnv, err := pkg.GenerateEnv(cfg, "", "")
		if err != nil {
			return "", fmt.Errorf("generate env: %w", err)
		}
		return defEnv, nil

	default:
		return "", fmt.Errorf("unsupported config format: %s", format)
	}
}
