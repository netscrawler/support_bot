-- Роли пользователей
CREATE TYPE user_role AS ENUM ('admin', 'user', 'primary');

-- Таблица пользователей
CREATE TABLE tg_users (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    role user_role NOT NULL DEFAULT 'user'
);

CREATE TABLE users (
   p_id SERIAL PRIMARY KEY,
   id varchar(256) unique NOT NULL,
   login VARCHAR(256) unique NOT NULL,
   email VARCHAR(256),
   password VARCHAR(256),
   role user_role NOT NULL DEFAULT 'user',
   tg_profile bigint,
   active boolean,
   constraint tg_profile foreign key (tg_profile) references tg_users(id)
);

CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
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

