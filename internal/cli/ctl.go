package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"support_bot/internal/config"
	"support_bot/internal/models"
	"support_bot/internal/postgres"
	"support_bot/internal/processor/lua"
	"support_bot/internal/repository"
	"support_bot/internal/service"
)

func Ctl() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s ctl <command> [args]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands: export, create, apply, validate, script\n")
		return
	}

	var err error
	switch os.Args[2] {
	case "export":
		err = exportReportsDslFromDB(os.Args[3:])
	case "create":
		err = createReportDSLTemplate(os.Args[3:])
	case "apply":
		err = applyDSLs(os.Args[3:])
	case "validate":
		err = validateDSLs(os.Args[3:])
	case "script":
		err = scriptCtl(os.Args[3:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown ctl command: %s\n", os.Args[2])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupDB(ctx context.Context, cfgPath string) (*postgres.DB, *config.Config, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}

	db, err := postgres.New(ctx, cfg.Database, slog.Default())
	if err != nil {
		return nil, nil, fmt.Errorf("create database connection: %w", err)
	}

	return db, cfg, nil
}

func exportReportsDslFromDB(args []string) error {
	fs := flag.NewFlagSet("ctl export", flag.ContinueOnError)

	cfgPath := fs.String("config", "", "Путь к конфигурационному файлу")
	out := fs.String("out", "", "Путь к папке для сохранения отчетов")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, _, err := setupDB(ctx, *cfgPath)
	if err != nil {
		return err
	}
	defer db.Stop(ctx)

	mng := service.NewReportManager(
		repository.NewRepository(db.GetConn(), slog.Default()),
		service.NewReportValidation(),
		slog.Default(),
	)

	reports, err := mng.Load(ctx)
	if err != nil {
		return fmt.Errorf("load reports from db: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Loaded %d reports from db\n", len(reports))

	for _, r := range reports {
		data, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			slog.Error("Unable to marshal report", "report", r.Name, "error", err)
			continue
		}

		fName := strings.ToLower(r.Name)
		rP := filepath.Join(*out, fName+".json")

		if err := os.MkdirAll(filepath.Dir(rP), 0o755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		if err := os.WriteFile(rP, data, 0o644); err != nil {
			slog.Error("Unable to save report", "path", rP, "error", err)
			continue
		}
		fmt.Fprintf(os.Stdout, "Saved report %s to %s\n", r.Name, rP)
	}

	return nil
}

func createReportDSLTemplate(args []string) error {
	fs := flag.NewFlagSet("ctl create", flag.ContinueOnError)

	out := fs.String("out", "", "Путь к файлу для сохранения шаблона")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var wr io.Writer

	if *out != "" {
		file, err := os.OpenFile(*out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		defer file.Close()
		wr = file
	} else {
		wr = os.Stdout
	}

	b, err := json.MarshalIndent(models.ReportExample, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report example: %w", err)
	}
	_, err = wr.Write(b)
	if err != nil {
		return fmt.Errorf("write report example: %w", err)
	}
	return nil
}

func applyDSLs(args []string) error {
	fs := flag.NewFlagSet("ctl apply", flag.ContinueOnError)

	cfgPath := fs.String("config", "", "Путь к конфигурационному файлу")
	dslsPath := fs.String("report", "", "Путь к файлу с отчетами")
	if err := fs.Parse(args); err != nil {
		return err
	}

	patterns := append([]string{}, fs.Args()...)
	if *dslsPath != "" {
		patterns = append([]string{*dslsPath}, patterns...)
	}

	if len(patterns) == 0 {
		return fmt.Errorf("no report path provided")
	}

	files, err := ResolveFiles(patterns...)
	if err != nil {
		return fmt.Errorf("resolve files: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Seen %d files: %v\n", len(files), files)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, _, err := setupDB(ctx, *cfgPath)
	if err != nil {
		return err
	}
	defer db.Stop(ctx)

	mng := service.NewReportManager(
		repository.NewRepository(db.GetConn(), slog.Default()),
		service.NewReportValidation(),
		slog.Default(),
	)

	for _, f := range files {
		if filepath.Ext(f) != ".json" {
			fmt.Fprintf(os.Stderr, "Skipping file %s: not a JSON file\n", f)
			continue
		}
		file, err := os.ReadFile(f)
		if err != nil {
			slog.Error("Unable to read file", "path", f, "error", err)
			continue
		}

		fmt.Fprintf(os.Stdout, "Saving report %s\n", f)
		err = mng.Create(ctx, bytes.NewReader(file))
		if err != nil {
			slog.Error("Unable to save report to db", "path", f, "error", err)
			continue
		}
		fmt.Fprintf(os.Stdout, "Saved report %s\n", f)
	}

	return nil
}

func ResolveFiles(patterns ...string) ([]string, error) {
	var files []string

	for _, pattern := range patterns {
		if strings.ContainsAny(pattern, "*?[]") {
			matched, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
			}
			files = append(files, matched...)
			continue
		}

		files = append(files, pattern)
	}

	return files, nil
}

func validateDSLs(args []string) error {
	fs := flag.NewFlagSet("ctl validate", flag.ContinueOnError)
	dslsPath := fs.String("report", "", "Путь к файлу с отчетами")
	if err := fs.Parse(args); err != nil {
		return err
	}

	patterns := append([]string{}, fs.Args()...)
	if *dslsPath != "" {
		patterns = append([]string{*dslsPath}, patterns...)
	}

	if len(patterns) == 0 {
		return fmt.Errorf("no report path provided")
	}

	files, err := ResolveFiles(patterns...)
	if err != nil {
		return fmt.Errorf("resolve files: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	validator := service.NewReportValidation()

	for _, f := range files {
		if filepath.Ext(f) != ".json" {
			continue
		}
		file, err := os.ReadFile(f)
		if err != nil {
			slog.Error("Unable to read file", "path", f, "error", err)
			continue
		}

		var report models.Report
		if err := json.Unmarshal(file, &report); err != nil {
			return fmt.Errorf("unmarshal report %s: %w", f, err)
		}

		if err := validator.Validate(ctx, report); err != nil {
			return fmt.Errorf("validate report %s: %w", f, err)
		}
		fmt.Fprintf(os.Stdout, "Report %s is valid\n", f)
	}

	return nil
}

func scriptCtl(args []string) error {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: ctl script <command> [args]\n")
		fmt.Fprintf(os.Stderr, "Commands: create, save\n")
		return nil
	}

	switch args[0] {
	case "create":
		return createExampleScript(args[1:])
	case "save":
		return saveScript(args[1:])
	default:
		return fmt.Errorf("unknown script command: %s", args[0])
	}
}

func saveScript(args []string) error {
	fs := flag.NewFlagSet("ctl script save", flag.ContinueOnError)

	cfgPath := fs.String("config", "", "Путь к конфигурационному файлу")
	scriptPath := fs.String("script", "", "Путь к файлу скрипта")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *scriptPath == "" {
		return fmt.Errorf("script path is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, _, err := setupDB(ctx, *cfgPath)
	if err != nil {
		return err
	}
	defer db.Stop(ctx)

	scriptName := filepath.Base(*scriptPath)

	script, err := os.ReadFile(*scriptPath)
	if err != nil {
		return fmt.Errorf("read script file: %w", err)
	}

	mng := service.NewScriptManager(repository.NewScript(db.GetConn()))

	err = mng.Save(ctx, scriptName, string(script))
	if err != nil {
		return fmt.Errorf("save script: %w", err)
	}
	return nil
}

func createExampleScript(args []string) error {
	fs := flag.NewFlagSet("ctl script create", flag.ContinueOnError)

	out := fs.String("out", "", "Путь к файлу для сохранения скрипта")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var wr io.Writer

	if *out != "" {
		file, err := os.OpenFile(*out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		defer file.Close()
		wr = file
	} else {
		wr = os.Stdout
	}

	if _, err := wr.Write([]byte(lua.ExampleLuaPlugin)); err != nil {
		return fmt.Errorf("write script: %w", err)
	}
	return nil
}
