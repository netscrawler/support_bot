-- name: GetScriptByName :one
select script from lua_scripts where name = $1;

-- name: SaveScriptIfAbsent :exec
insert into lua_scripts(name, script) values ($1, $2)
on conflict (name) do nothing;
