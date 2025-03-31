package repository

import (
	"context"
	"errors"
	"fmt"
	"support_bot/internal/database/postgres"
	"support_bot/internal/models"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type User struct {
	storage *postgres.Storage
	log     *zap.Logger
}

func NewUser(s *postgres.Storage, log *zap.Logger) User {
	return User{
		storage: s,
		log:     log,
	}
}

func (u *User) Update(ctx context.Context, usr *models.User) error {
	const op = "repository.User.Update"
	query, args, err := u.storage.Builder.
		Update("users").
		Set("telegram_id", usr.TelegramID).
		Set("first_name", usr.FirstName).
		Set("last_name", usr.LastName).
		Where(squirrel.Eq{"username": usr.Username}).
		ToSql()
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))
		return err
	}

	_, err = u.storage.Db.Exec(ctx, query, args...)
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error exec query: %s", op, err.Error()))
		return err
	}

	return nil
}

func (u *User) Create(ctx context.Context, usr *models.User) error {
	const op = "repository.User.Create"
	query, args, err := u.storage.Builder.
		Insert("users").
		Columns(
			"telegram_id",
			"username",
			"first_name",
			"last_name",
			"role",
		).
		Values(
			usr.TelegramID,
			usr.Username,
			usr.FirstName,
			usr.LastName,
			usr.Role,
		).
		ToSql()
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))
		return err
	}

	_, err = u.storage.Db.Exec(ctx, query, args...)
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error exec query: %s", op, err.Error()))
		return err
	}

	return nil
}

func (u *User) GetByUsername(ctx context.Context, uname string) (*models.User, error) {
	const op = "repository.User.GetByTgId"

	query, args, err := u.storage.Builder.
		Select(
			"id",
			"telegram_id",
			"username",
			"first_name",
			"last_name",
			"role",
		).
		From("users").
		Where(squirrel.Eq{"username": uname}).
		ToSql()
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))

		return nil, fmt.Errorf("%s : %w", op, err)
	}
	var user models.User
	row := u.storage.Db.QueryRow(ctx, query, args...)
	err = row.Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.Role,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			u.log.Error(fmt.Sprintf("%s not found user with username: %s", op, uname))
			return nil, models.ErrNotFound
		}
		u.log.Error(fmt.Sprintf("%s | %s", op, err.Error()))
		return nil, err
	}

	return &user, nil
}

func (u *User) GetByTgId(ctx context.Context, id int64) (*models.User, error) {
	const op = "repository.User.GetByTgId"

	query, args, err := u.storage.Builder.
		Select(
			"id",
			"telegram_id",
			"username",
			"first_name",
			"last_name",
			"role",
		).
		From("users").
		Where(squirrel.Eq{"telegram_id": id}).
		ToSql()
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))

		return nil, fmt.Errorf("%s : %w", op, err)
	}
	var user models.User
	row := u.storage.Db.QueryRow(ctx, query, args...)
	err = row.Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.Role,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			u.log.Info(fmt.Sprintf("%s not found user with id:%d", op, id))
			return nil, models.ErrNotFound
		}
		u.log.Error(fmt.Sprintf("%s | %s", op, err.Error()))
		return nil, err
	}

	return &user, nil
}

func (u *User) GetAllAdmins(ctx context.Context) ([]models.User, error) {
	const op = "repository.User.GetAllAdmins"
	query, args, err := u.storage.Builder.
		Select(
			"id",
			"telegram_id",
			"username",
			"first_name",
			"last_name",
			"role",
		).
		From("users").
		Where(squirrel.Eq{"role": models.AdminRole}).
		ToSql()
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))

		return nil, fmt.Errorf("%s : %w", op, err)
	}

	rows, err := u.storage.Db.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			u.log.Error(fmt.Sprintf("%s | %s", op, err))
			return nil, models.ErrNotFound
		}
		u.log.Error(op, zap.Error(err))
		return nil, fmt.Errorf("%s : %w", op, err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.Role,
		); err != nil {
			u.log.Error(fmt.Sprintf("%s | %s", op, err.Error()))
			continue
		}
		users = append(users, user)
	}
	u.log.Info(fmt.Sprintf("%s : successfully got %d users", op, len(users)))
	return users, nil
}

func (u *User) GetAll(ctx context.Context) ([]models.User, error) {
	const op = "repository.User.GetAll"
	query, args, err := u.storage.Builder.
		Select(
			"id",
			"telegram_id",
			"username",
			"first_name",
			"last_name",
			"role",
		).
		From("users").
		ToSql()
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))

		return nil, fmt.Errorf("%s : %w", op, err)
	}

	rows, err := u.storage.Db.Query(ctx, query, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			u.log.Error(fmt.Sprintf("%s | %s", op, err))
			return nil, models.ErrNotFound
		}
		u.log.Error(op, zap.Error(err))
		return nil, fmt.Errorf("%s : %w", op, err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.Role,
		); err != nil {
			u.log.Error(fmt.Sprintf("%s | %s", op, err.Error()))
			continue
		}
		users = append(users, user)
	}
	u.log.Info(fmt.Sprintf("%s : successfully got %d users", op, len(users)))
	return users, nil
}

func (u *User) Delete(ctx context.Context, tgId int64) error {
	const op = "repository.User.Delete"

	query, args, err := u.storage.Builder.
		Delete("users").
		Where(squirrel.Eq{"telegram_id": tgId}).
		ToSql()
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error building query: %s", op, err.Error()))
		return err
	}

	_, err = u.storage.Db.Exec(ctx, query, args...)
	if err != nil {
		u.log.Error(fmt.Sprintf("%s error exec query: %s", op, err.Error()))
		return err
	}

	return nil
}
