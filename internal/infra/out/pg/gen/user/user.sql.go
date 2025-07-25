// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: user.sql

package userrepo

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createUser = `-- name: CreateUser :one
INSERT INTO users (
    telegram_id, username, first_name, last_name, role
) VALUES ( $1,$2,$3,$4, $5)
RETURNING id, telegram_id, username, first_name, last_name, role
`

type CreateUserParams struct {
	TelegramID int64
	Username   pgtype.Text
	FirstName  pgtype.Text
	LastName   pgtype.Text
	Role       UserRole
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, createUser,
		arg.TelegramID,
		arg.Username,
		arg.FirstName,
		arg.LastName,
		arg.Role,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.TelegramID,
		&i.Username,
		&i.FirstName,
		&i.LastName,
		&i.Role,
	)
	return i, err
}

const deleteUserbyTgID = `-- name: DeleteUserbyTgID :exec
DELETE FROM users
    WHERE telegram_id = $1
`

func (q *Queries) DeleteUserbyTgID(ctx context.Context, telegramID int64) error {
	_, err := q.db.Exec(ctx, deleteUserbyTgID, telegramID)
	return err
}

const getAllAdmins = `-- name: GetAllAdmins :many
SELECT id, telegram_id, username, first_name, last_name, role FROM users
WHERE role in ('admin', 'primary')
`

func (q *Queries) GetAllAdmins(ctx context.Context) ([]User, error) {
	rows, err := q.db.Query(ctx, getAllAdmins)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.TelegramID,
			&i.Username,
			&i.FirstName,
			&i.LastName,
			&i.Role,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getAllUsers = `-- name: GetAllUsers :many
SELECT id, telegram_id, username, first_name, last_name, role FROM users
`

func (q *Queries) GetAllUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.Query(ctx, getAllUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.TelegramID,
			&i.Username,
			&i.FirstName,
			&i.LastName,
			&i.Role,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserByTgID = `-- name: GetUserByTgID :one
SELECT id, telegram_id, username, first_name, last_name, role FROM users
WHERE telegram_id = $1
LIMIT 1
`

func (q *Queries) GetUserByTgID(ctx context.Context, telegramID int64) (User, error) {
	row := q.db.QueryRow(ctx, getUserByTgID, telegramID)
	var i User
	err := row.Scan(
		&i.ID,
		&i.TelegramID,
		&i.Username,
		&i.FirstName,
		&i.LastName,
		&i.Role,
	)
	return i, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT id, telegram_id, username, first_name, last_name, role FROM users
WHERE username = $1
LIMIT 1
`

func (q *Queries) GetUserByUsername(ctx context.Context, username pgtype.Text) (User, error) {
	row := q.db.QueryRow(ctx, getUserByUsername, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.TelegramID,
		&i.Username,
		&i.FirstName,
		&i.LastName,
		&i.Role,
	)
	return i, err
}

const updateUser = `-- name: UpdateUser :exec
UPDATE users
    SET telegram_id = $2,
        first_name = $3,
        last_name = $4
    WHERE username = $1
`

type UpdateUserParams struct {
	Username   pgtype.Text
	TelegramID int64
	FirstName  pgtype.Text
	LastName   pgtype.Text
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) error {
	_, err := q.db.Exec(ctx, updateUser,
		arg.Username,
		arg.TelegramID,
		arg.FirstName,
		arg.LastName,
	)
	return err
}
