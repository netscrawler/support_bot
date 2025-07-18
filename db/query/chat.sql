-- name: CreateChat :one
INSERT INTO chats (
    chat_id, title, type, description, is_active
) VALUES ( $1,$2,$3,$4,$5 )
RETURNING *;

-- name: GetChatByTitle :one
Select * from chats
where title =$1
Limit 1;

-- name: GetAllChats :many
SELECT * FROM chats;

-- name: DeleteChatByID :exec
DELETE FROM chats
    WHERE chat_id = $1;
