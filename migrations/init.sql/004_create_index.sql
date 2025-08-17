
-- Индексы
CREATE INDEX idx_notify_chat_id ON notify(chat_id);
CREATE INDEX idx_notify_group_id ON notify(group_id);
CREATE INDEX idx_notify_query_notify_id ON notify_query(notify_id);
