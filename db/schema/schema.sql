-- Роли пользователей
CREATE TYPE user_role AS ENUM ('admin', 'user', 'primary');

-- Таблица пользователей
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    role user_role NOT NULL DEFAULT 'user'
);

-- Таблица чатов для уведомлений
CREATE TABLE chats (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT UNIQUE NOT NULL,
    title VARCHAR(255),
    type VARCHAR(50) NOT NULL, -- 'private', 'group', 'supergroup', 'channel'
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Группы уведомлений
CREATE TABLE notify_groups (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE, -- уникальный идентификатор группы
    title TEXT NOT NULL
);

-- Запросы для уведомлений
CREATE TABLE queries (
    id SERIAL PRIMARY KEY,
    card_uuid TEXT NOT NULL,
    title TEXT
);

CREATE TABLE templates (
    id SERIAL PRIMARY KEY,
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
    query_id INT,    -- связь с запросом (один к одному)
    template_id INT,    -- связь с запросом (один к одному)
    CONSTRAINT fk_notify_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE RESTRICT,
    CONSTRAINT fk_notify_group FOREIGN KEY (group_id) REFERENCES notify_groups(id) ON DELETE SET NULL,
    CONSTRAINT fk_notify_query FOREIGN KEY (query_id) REFERENCES queries(id) ON DELETE CASCADE,
    CONSTRAINT fk_notify_template FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE
);

-- Индексы
CREATE INDEX idx_notify_chat_id ON notify(chat_id);
CREATE INDEX idx_notify_group_id ON notify(group_id);


-- Триггер: при удалении чата уведомления становятся неактивными
CREATE OR REPLACE FUNCTION deactivate_notify_on_chat_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE notify SET active = FALSE WHERE chat_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_deactivate_notify_on_chat_delete
BEFORE DELETE ON chats
FOR EACH ROW
EXECUTE FUNCTION deactivate_notify_on_chat_delete();

