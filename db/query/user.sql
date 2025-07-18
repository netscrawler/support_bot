-- name: CreateUser :one
INSERT INTO users (
    telegram_id, username, first_name, last_name, role
) VALUES ( $1,$2,$3,$4, $5)
RETURNING *;

-- name: UpdateUser :exec
UPDATE users
    SET telegram_id = $2,
        first_name = $3,
        last_name = $4
    WHERE username = $1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1
LIMIT 1;

-- name: GetUserByTgID :one
SELECT * FROM users
WHERE telegram_id = $1
LIMIT 1;

-- name: GetAllAdmins :many
SELECT * FROM users
WHERE role in ('admin', 'primary');

-- name: GetAllUsers :many
SELECT * FROM users;

-- name: DeleteUserbyTgID :exec
DELETE FROM users
    WHERE telegram_id = $1;
