package postgres

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type Storage struct {
	log     *zap.Logger
	DB      *pgx.Conn
	Builder *squirrel.StatementBuilderType
}

func New(log *zap.Logger) *Storage {
	builder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	return &Storage{log: log, Builder: &builder}
}

func (s *Storage) Init(ctx context.Context, connStr string) error {
	const op = "storage.postgres.Init"

	var err error

	connCfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		s.log.Fatal("%s unable to ParseConfig")

		return err
	}

	s.DB, err = pgx.ConnectConfig(ctx, connCfg)
	if err != nil {
		s.log.Error(op, zap.Error(err))

		return fmt.Errorf("%s : %w", op, err)
	}

	if err := s.DB.Ping(ctx); err != nil {
		s.log.Error(op, zap.Error(err))

		return fmt.Errorf("%s : %w", op, err)
	}

	s.log.Info(op + " : successfully connected")

	return nil
}

func (s *Storage) Close(ctx context.Context) {
	s.DB.Close(ctx)
}
