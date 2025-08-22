-- name: ListAllNotifies :many
SELECT
    n.id,
    n.name,
    ng.name  AS group_id,
    nq.card_uuid,
    n.cron,
    tp.template_text,
    n.title,
    ng.title AS group_title,
    c.chat_id,
    n.active,
    n.format,
    n.thread_id
FROM notify n
LEFT JOIN chats c ON n.chat_id = c.id
LEFT JOIN notify_groups ng ON n.group_id = ng.id
LEFT JOIN queries nq ON nq.id = n.query_id
LEFT JOIN templates tp ON tp.id = n.template_id
ORDER BY n.id;

-- name: ListAllActiveNotifies :many
SELECT
    n.id,
    n.name,
    ng.name  AS group_id,
    nq.card_uuid,
    n.cron,
    tp.template_text,
    n.title,
    ng.title AS group_title,
    c.chat_id,
    n.active,
    n.format,
	n.thread_id

FROM notify n
LEFT JOIN chats c ON n.chat_id = c.id
LEFT JOIN notify_groups ng ON n.group_id = ng.id
LEFT JOIN queries nq ON nq.id = n.query_id
LEFT JOIN templates tp ON tp.id = n.template_id
WHERE n.active = TRUE
ORDER BY n.id;

