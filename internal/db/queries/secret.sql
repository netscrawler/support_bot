-- name: UpsertSecret :exec
insert into secrets(name, value, host) values ($1, $2, $3)
on conflict (name) do update
set value = excluded.value, host = excluded.host, updated_at = now();

-- name: GetSecret :one
select value, host from secrets where name = $1;

-- name: ListSecrets :many
select name, host from secrets order by name;

-- name: DeleteSecret :execrows
delete from secrets where name = $1;
