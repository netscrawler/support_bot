-- chat.sql
-- Phase 1 only adds the two queries ReportStore.Create's transaction needs
-- to resolve/create a chat by chat_id. Full chat CRUD (GetByTitle, ListChats,
-- DeleteChat) is Phase 2's job — do not add them here.

-- name: FindChatIDByChatID :one
select id from chats where chat_id = $1;

-- name: CreateChat :one
insert into chats(chat_id, title, type, description, is_active, ch_type)
values ($1, $2, $3, $4, $5, $6)
returning id;
