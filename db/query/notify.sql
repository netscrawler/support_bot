-- name: ListAllNotifies :many
SELECT	*
FROM notify;

-- name: ListAllActiveNotifies :many
SELECT	*
FROM notify
WHERE active = TRUE;
