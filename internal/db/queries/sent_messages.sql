-- sent_messages.sql
-- Запросы для таблицы sent_messages, используемые SentMsgStore
-- (internal/orchestrator.Deleter/Orchestrator). Переносит SQL из
-- internal/orchestrator/result_repo.go (SentMsgRepository, фаза 3
-- sqlc-миграции) без изменения текста запросов.

-- name: InsertSentMessage :exec
insert into sent_messages(chat_id, thread_id, message_id, message_id_str, title, sent_at, report_name, ch_type)
values ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: LoadSentMsgsToDeleteForUpdate :many
select id, chat_id, thread_id, message_id, message_id_str, title, sent_at, deleted, ch_type
from sent_messages
where deleted = false
  and sent_at >= CURRENT_DATE - INTERVAL '1 day'
  and sent_at < CURRENT_DATE
for update skip locked;

-- name: MarkSentMsgDeleted :exec
update sent_messages set deleted = true where id = $1;

-- name: DeleteAllMarkedSentMessages :execrows
delete from sent_messages where deleted = true;

-- name: MarkEndOfDayMsgDeleted :exec
UPDATE sent_messages sm
SET deleted = TRUE
FROM (
    SELECT DISTINCT ON (report_name, chat_id)
        id
    FROM sent_messages
    WHERE deleted = FALSE
      AND sent_at >= CURRENT_DATE - INTERVAL '1 day'
      AND sent_at < CURRENT_DATE
    ORDER BY report_name, chat_id, sent_at DESC, id DESC
) last_msgs
WHERE sm.id = last_msgs.id;
