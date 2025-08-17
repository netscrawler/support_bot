-- Группы уведомлений
CREATE TABLE notify_groups (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE, -- уникальный идентификатор группы
    title TEXT NOT NULL
);
-- Запросы для уведомлений
CREATE TABLE notify_query (
    id SERIAL PRIMARY KEY,
    card_uuid TEXT NOT NULL,
    template_text TEXT,
    title TEXT
);

-- Уведомления
CREATE TABLE notify (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE, -- уникальное имя уведомления
    cron TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    format TEXT[],
    title TEXT NOT NULL,
    thread_id BIGINT NOT NULL DEFAULT 0,
    chat_id INT NOT NULL,   -- связь с чатом
    group_id INT,           -- связь с группой (опционально)
    query_id INT UNIQUE,    -- связь с запросом (один к одному)
    CONSTRAINT fk_notify_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE RESTRICT,
    CONSTRAINT fk_notify_group FOREIGN KEY (group_id) REFERENCES notify_groups(id) ON DELETE SET NULL,
    CONSTRAINT fk_notify_query FOREIGN KEY (query_id) REFERENCES notify_query(id) ON DELETE CASCADE
);

