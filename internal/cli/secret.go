package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"support_bot/internal/config"
	"support_bot/internal/pkg/secret"
	"support_bot/internal/store"
)

const (
	secretSet    = "set"
	secretList   = "list"
	secretDelete = "delete"
)

func secretCtl(args []string) error {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: ctl secret <command> [args]\n")
		fmt.Fprintf(os.Stderr, "Commands: set, list, delete\n")

		return nil
	}

	cmd := args[0]
	if cmd != secretSet && cmd != secretList && cmd != secretDelete {
		return fmt.Errorf("unknown secret command: %s", cmd)
	}

	fs := flag.NewFlagSet("ctl secret "+cmd, flag.ContinueOnError)
	cfgPath := fs.String("config", "", "Путь к конфигурационному файлу")
	name := fs.String("name", "", "Имя секрета")
	host := fs.String(
		"host",
		"",
		"Хост[:порт], на который разрешено отправлять секрет (для set)",
	)

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if cmd != secretList && *name == "" {
		return errors.New("no secret name provided")
	}

	if cmd == secretSet && *host == "" {
		return errors.New("no secret host provided")
	}

	var value string

	if cmd == secretSet {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read secret from stdin: %w", err)
		}

		value = strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
		if value == "" {
			return errors.New("empty secret value")
		}
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var c *secret.Cipher

	if cfg.Secrets.Key != "" {
		if c, err = secret.NewCipher(cfg.Secrets.Key.Value()); err != nil {
			return err
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	db, err := openDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeDB(ctx, db)

	st := store.NewSecretStore(db.GetConn(), c)

	switch cmd {
	case secretSet:
		if err := st.Set(ctx, *name, *host, value); err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "Secret %s saved\n", *name)

	case secretList:
		list, err := st.List(ctx)
		if err != nil {
			return err
		}

		for _, s := range list {
			fmt.Fprintf(os.Stdout, "%s\t%s\n", s.Name, s.Host)
		}

	case secretDelete:
		if err := st.Delete(ctx, *name); err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "Secret %s deleted\n", *name)
	}

	return nil
}
