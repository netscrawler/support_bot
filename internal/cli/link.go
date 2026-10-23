package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"support_bot/internal/models"
	"support_bot/internal/store"
	"text/tabwriter"
	"time"
)

type linkStore interface {
	CreatePublicLink(ctx context.Context, reportName string, ttl time.Duration) (string, error)
	ListPublicLinks(ctx context.Context, reportName string) ([]models.PublicLink, error)
	ExpirePublicLink(ctx context.Context, publicID string) (bool, error)
}

// openLinkStore подключает БД; возвращает store и функцию закрытия.
type openLinkStore func(ctx context.Context, cfgPath string) (linkStore, func(), error)

func openLinkStoreDB(ctx context.Context, cfgPath string) (linkStore, func(), error) {
	db, _, err := setupDB(ctx, cfgPath)
	if err != nil {
		return nil, nil, err
	}
	return store.NewReportStore(db.GetConn(), slog.Default()), func() { _ = db.Stop(ctx) }, nil
}

func linkCtl(args []string) error {
	if len(args) < 1 {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: ctl link <command> [args]\n")
		_, _ = fmt.Fprintf(os.Stderr, "Commands: create, list, revoke\n")
		return nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	switch args[0] {
	case "create":
		return linkCreate(ctx, openLinkStoreDB, os.Stdout, args[1:])
	case "list":
		return linkList(ctx, openLinkStoreDB, os.Stdout, args[1:])
	case "revoke":
		return linkRevoke(ctx, openLinkStoreDB, os.Stdout, args[1:])
	default:
		return fmt.Errorf("unknown link command: %s", args[0])
	}
}

func linkCreate(ctx context.Context, open openLinkStore, w io.Writer, args []string) error {
	fs := flag.NewFlagSet("ctl link create", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "Путь к конфигурационному файлу")
	report := fs.String("report", "", "Имя отчёта")
	ttl := fs.Duration("ttl", 0, "Срок жизни ссылки, например 168h (0 — бессрочно)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *report == "" {
		return errors.New("no report name provided")
	}

	st, closeFn, err := open(ctx, *cfgPath)
	if err != nil {
		return err
	}
	defer closeFn()

	id, err := st.CreatePublicLink(ctx, *report, *ttl)
	if err != nil {
		return fmt.Errorf("create link: %w", err)
	}
	_, _ = fmt.Fprintf(w, "%s\n/api/v1/public/report/%s\n", id, id)
	return nil
}

func linkList(ctx context.Context, open openLinkStore, w io.Writer, args []string) error {
	fs := flag.NewFlagSet("ctl link list", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "Путь к конфигурационному файлу")
	report := fs.String("report", "", "Фильтр по имени отчёта")
	if err := fs.Parse(args); err != nil {
		return err
	}

	st, closeFn, err := open(ctx, *cfgPath)
	if err != nil {
		return err
	}
	defer closeFn()

	links, err := st.ListPublicLinks(ctx, *report)
	if err != nil {
		return fmt.Errorf("list links: %w", err)
	}

	now := time.Now()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "UUID\tОТЧЁТ\tСОЗДАНА\tИСТЕКАЕТ\tСОСТОЯНИЕ")
	for _, l := range links {
		expires, state := "-", "активна"
		if l.ExpiresAt != nil {
			expires = l.ExpiresAt.Format(time.RFC3339)
		}
		if !l.Active(now) {
			state = "истекла"
		}
		_, _ = fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\n",
			l.ID,
			l.Report,
			l.CreatedAt.Format(time.RFC3339),
			expires,
			state,
		)
	}
	return tw.Flush()
}

func linkRevoke(ctx context.Context, open openLinkStore, w io.Writer, args []string) error {
	fs := flag.NewFlagSet("ctl link revoke", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "Путь к конфигурационному файлу")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: ctl link revoke -config <путь> <uuid>")
	}

	st, closeFn, err := open(ctx, *cfgPath)
	if err != nil {
		return err
	}
	defer closeFn()

	revoked, err := st.ExpirePublicLink(ctx, fs.Arg(0))
	switch {
	case errors.Is(err, models.ErrNotFound):
		return errors.New("ссылка не найдена")
	case err != nil:
		return fmt.Errorf("revoke link: %w", err)
	case revoked:
		_, _ = fmt.Fprintln(w, "ссылка отозвана")
	default:
		_, _ = fmt.Fprintln(w, "ссылка уже неактивна")
	}
	return nil
}
