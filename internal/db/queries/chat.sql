-- chat.sql
-- CRUD-запросы для таблицы chats, используемые ChatStore и
-- ReportStore.Create (FindChatIDByChatID/CreateChat, добавлены в фазе 1).

-- name: FindChatIDByChatID :one
select id from chats where chat_id = $1;

-- name: CreateChat :one
insert into chats(chat_id, title, type, description, is_active, ch_type)
values ($1, $2, $3, $4, $5, $6)
returning id;

-- name: GetChatByTitle :one
select id, chat_id, title, type, description, is_active, ch_type
from chats
where title = $1
limit 1;

-- name: ListChats :many
select id, chat_id, title, type, description, is_active, ch_type
from chats
where is_active = false;

-- name: DeleteChat :exec
delete from chats where chat_id = $1;
