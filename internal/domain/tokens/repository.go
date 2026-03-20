package tokens

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"support_bot/internal/errorz"
	"support_bot/internal/postgres"

	"github.com/jmoiron/sqlx"
)

type TokenRepository struct {
	db *sqlx.DB
	*postgres.Uow
}

func NewTokenRepository(db *sqlx.DB) *TokenRepository {
	tr := &TokenRepository{db: db}
	uow := postgres.NewUOW(tr.getDB)
	tr.Uow = uow
	return tr
}

func (tr *TokenRepository) Start(ctx context.Context) (context.Context, error) {
	return postgres.Start(ctx, tr.db)
}

func (tr *TokenRepository) Commit(ctx context.Context) error {
	tx, err := tr.TxFromCtx(ctx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (tr *TokenRepository) Rollback(ctx context.Context) error {
	tx, err := tr.TxFromCtx(ctx)
	if err != nil {
		return err
	}

	return tx.Rollback()
}

func (tr *TokenRepository) Save(ctx context.Context, token Token) error {
	const stmt = `INSERT INTO refresh_tokens (user_id, token_hash, revoked, created_at, expires_at) VALUES ($1, $2, $3, $4, $5)`

	tx, err := tr.TxFromCtx(ctx)
	if err != nil {
		return fmt.Errorf("%w : %w", errorz.ErrInternal, err)
	}

	_, err = tx.ExecContext(ctx, stmt, token.UserID, token.TokenHash, token.Revoked, token.CreatedAt, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf(" %w : %w", errorz.ErrInternal, err)
	}

	return nil
}

func (tr *TokenRepository) RevokeTokensByUserID(ctx context.Context, userID string) (int64, error) {
	const stmt = `UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`

	tx, err := tr.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, fmt.Errorf(" %w : %w", errorz.ErrInternal, err)
	}

	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, stmt, userID)
	if err != nil {
		return 0, fmt.Errorf(" %w : %w", errorz.ErrInternal, err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf(" %w : %w", errorz.ErrInternal, err)
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf(" %w : %w", errorz.ErrInternal, err)
	}

	return count, nil
}

func (tr *TokenRepository) FindActiveByUserID(ctx context.Context, userID string) ([]TokenDBO, error) {
	const stmt = `select * from refresh_tokens where user_id = $1 and revoked = false`

	tx, err := tr.db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf(" %w : %w", errorz.ErrInternal, err)
	}
	defer tx.Rollback()

	var token []TokenDBO

	if err := tx.SelectContext(ctx, &token, stmt, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorz.ErrNotFound
		}

		return nil, err
	}

	tx.Commit()
	return token, nil
}

func (tr *TokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (TokenDBO, error) {
	const stmt = `select * from refresh_tokens where token_hash = $1 limit 1`

	var token TokenDBO

	if err := tr.db.GetContext(ctx, &token, stmt, tokenHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return token, errorz.ErrNotFound
		}

		return token, err
	}
	return token, nil
}

func (tr *TokenRepository) getDB() *sqlx.DB {
	return tr.db
}
