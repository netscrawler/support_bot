CREATE TYPE user_role AS ENUM ('admin', 'user', 'primary');
CREATE TYPE notify_format AS ENUM ('text', 'png', 'csv', 'xlsx');
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

CREATE TABLE notify (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL, --Имя уведомления
    group_id TEXT, -- Идентификатор группы
    card_uuid TEXT NOT NULL,
    cron TEXT NOT NULL,
    template_text TEXT,
    title TEXT,
    group_title TEXT,
    chat_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    format TEXT[]
);
