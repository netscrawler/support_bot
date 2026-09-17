-- user.sql
-- CRUD-запросы для таблицы users, используемые UserStore.

-- name: CreateUser :one
insert into users(telegram_id, username, first_name, last_name, role)
values ($1, $2, $3, $4, $5)
returning id;

-- name: UpdateUser :exec
update users
set telegram_id = $2,
    first_name  = $3,
    last_name   = $4
where username = $1;

-- name: GetUserByUsername :one
select id, telegram_id, username, first_name, last_name, role
from users
where username = $1
limit 1;

-- name: ListUsers :many
select id, telegram_id, username, first_name, last_name, role
from users;

-- name: GetUserByTelegramID :one
select id, telegram_id, username, first_name, last_name, role
from users
where telegram_id = $1
limit 1;

-- name: ListAdmins :many
select id, telegram_id, username, first_name, last_name, role
from users
where role in ('admin', 'primary');

-- name: DeleteUser :exec
delete from users where telegram_id = $1;
