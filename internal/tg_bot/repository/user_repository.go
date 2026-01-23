package repository

import (
	"context"
	"fmt"
	"log/slog"

	models "support_bot/internal/models/notify"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewUserRepository(db *sqlx.DB, log *slog.Logger) *UserRepository {
	l := log.With(slog.Any("module", "tg_bot.repository.user"))
	return &UserRepository{db: db, log: l}
}

func (u *UserRepository) Create(ctx context.Context, user *models.User) error {
	const query = `INSERT INTO users (
    telegram_id, username, first_name, last_name, role
) VALUES ( $1,$2,$3,$4, $5)
RETURNING *;`

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("user repository create: %w", err)
	}
	_, err := u.db.ExecContext(
		ctx,
		query,
		user.TelegramID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.Role,
	)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

func (u *UserRepository) Update(ctx context.Context, user *models.User) error {
	const query = `UPDATE users
    SET telegram_id = $2,
        first_name = $3,
        last_name = $4
    WHERE username = $1;`
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("user repository update: %w", err)
	}

	res, err := u.db.ExecContext(
		ctx,
		query,
		user.Username,
		user.TelegramID,
		user.FirstName,
		user.LastName,
	)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if count, err := res.RowsAffected(); err == nil {
		u.log.DebugContext(ctx, "updated users", slog.Any("count", count))
	}
	return nil
}

func (u *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	const query = `SELECT * FROM users
WHERE username = $1
LIMIT 1;`
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("user repository get by username: %w", err)
	}

	var user *models.User

	err := u.db.GetContext(ctx, user, query, username)
	if err != nil {
		return nil, fmt.Errorf("get by username: %w", err)
	}
	return user, nil
}

func (u *UserRepository) GetAll(ctx context.Context) ([]models.User, error) {
	const query = `SELECT * FROM users;`
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("user repository get all: %w", err)
	}

	var users []models.User
	if err := u.db.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("get all: %w", err)
	}
	return users, nil
}

func (u *UserRepository) GetByTgID(ctx context.Context, id int64) (*models.User, error) {
	const query = `SELECT * FROM users
WHERE telegram_id = $1
LIMIT 1;`
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("user repository get by telegram id: %w", err)
	}

	user := &models.User{}

	err := u.db.GetContext(ctx, user, query, id)
	if err != nil {
		return nil, fmt.Errorf("get by telegram id: %w", err)
	}
	return user, nil
}

func (u *UserRepository) GetAllAdmins(ctx context.Context) ([]models.User, error) {
	const query = `SELECT * FROM users
WHERE role in ('admin', 'primary');`
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("user repository get all admins: %w", err)
	}

	var users []models.User
	if err := u.db.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("get all admins: %w", err)
	}
	return users, nil
}

func (u *UserRepository) Delete(ctx context.Context, tgID int64) error {
	const query = `DELETE FROM users
    WHERE telegram_id = $1;`
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("user repository delete: %w", err)
	}

	res, err := u.db.ExecContext(ctx, query, tgID)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if count, err := res.RowsAffected(); err == nil {
		u.log.DebugContext(ctx, "deleted users", slog.Any("count", count))
	}
	return nil
}
