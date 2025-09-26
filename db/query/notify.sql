-- name: ListAllNotifies :many
SELECT
    n.id,
    n.name,
    ng.name  AS group_id,
array_agg(DISTINCT nq.card_uuid)::text[] AS card_uuids,
    n.cron,
    tp.template_text,
    n.title,
    ng.title AS group_title,
    n.remote_path,
    c.chat_id,
    n.active,
    n.format,
    n.thread_id
FROM notify n
LEFT JOIN chats c ON n.chat_id = c.id
LEFT JOIN notify_groups ng ON n.group_id = ng.id
LEFT JOIN notify_queries nqj ON nqj.notify_id = n.id
LEFT JOIN queries nq ON nq.id = nqj.query_id
LEFT JOIN templates tp ON tp.id = n.template_id
GROUP BY n.id, n.name, ng.name, n.cron, tp.template_text,
         n.title, ng.title, n.remote_path, c.chat_id,
         n.active, n.format, n.thread_id
ORDER BY n.id;


-- name: ListAllActiveNotifies :many
SELECT
    n.id,
    n.name,
    ng.name  AS group_id,
array_agg(DISTINCT nq.card_uuid)::text[] AS card_uuids,
    n.cron,
    tp.template_text,
    n.title,
    ng.title AS group_title,
    c.chat_id,
    n.remote_path,
    n.active,
    n.format,
    n.thread_id
FROM notify n
LEFT JOIN chats c ON n.chat_id = c.id
LEFT JOIN notify_groups ng ON n.group_id = ng.id
LEFT JOIN notify_queries nqj ON nqj.notify_id = n.id
LEFT JOIN queries nq ON nq.id = nqj.query_id
LEFT JOIN templates tp ON tp.id = n.template_id
WHERE n.active = TRUE
GROUP BY n.id, n.name, ng.name, n.cron, tp.template_text,
         n.title, ng.title, c.chat_id, n.remote_path,
         n.active, n.format, n.thread_id
ORDER BY n.id;

