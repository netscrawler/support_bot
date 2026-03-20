package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"support_bot/internal/domain/errorz"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (ur *UserRepository) GetByName(ctx context.Context, name string) (UserDBO, error) {
	const stmt = `select id, login, email, password, role  from users where login = $1 limit 1`

	if ctx.Err() != nil {
		return UserDBO{}, ctx.Err()
	}

	var u UserDBO

	err := ur.db.GetContext(ctx, &u, stmt, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return u, errorz.ErrUserNotFound
		}
		return u, fmt.Errorf("%w : (get user by name: %w)", errorz.ErrInternalServer, err)
	}

	return u, nil
}

func (ur *UserRepository) GetByID(ctx context.Context, userID string) (UserDBO, error) {
	const stmt = `select id, login, email, password, role  from users where id = $1 limit 1`

	if ctx.Err() != nil {
		return UserDBO{}, ctx.Err()
	}

	var u UserDBO

	err := ur.db.GetContext(ctx, &u, stmt, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return u, errorz.ErrUserNotFound
		}
		return u, fmt.Errorf("%w : (get user by name: %w)", errorz.ErrInternalServer, err)
	}

	return u, nil
}
