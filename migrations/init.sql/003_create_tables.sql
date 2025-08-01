CREATE TABLE notify (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    group_id TEXT,
    card_uuid TEXT NOT NULL,
    cron TEXT NOT NULL,
    template_text TEXT,
    title TEXT,
    group_title TEXT,
    chat_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    format TEXT[]
);

