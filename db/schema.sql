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

create table evaluate(
    id serial primary key,
    expr text NOT NULL
);

CREATE TABLE reports (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE, -- уникальное имя уведомления
    active BOOLEAN NOT NULL DEFAULT FALSE,
    title TEXT NOT NULL,
    eval_id bigint NOT NULL,
    CONSTRAINT fk_report_eval FOREIGN KEY (eval_id) REFERENCES evaluate(id) ON DELETE RESTRICT
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

create table email_templates(
    id serial PRIMARY KEY,
    Dest text[] NOT NULL,
    Copy text[],
    Subject text NOT NULL,
    Body text
);

create table recipients(
    id serial PRIMARY KEY,
    name text not null,
    config jsonb DEFAULT '{}',
    remote_path text,
    chat_id int,
    thread_id int,
    email_id int,
    type text,
    CONSTRAINT fk_recipient_chat FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE RESTRICT,
    CONSTRAINT fk_recipient_email FOREIGN KEY (email_id) REFERENCES email_templates(id) ON DELETE RESTRICT
);


CREATE TABLE reports_recipients(
    report_id int,
    recipient_id int,
    CONSTRAINT fk_reports_recipients_recipient FOREIGN KEY (recipient_id) REFERENCES recipients (id),
    CONSTRAINT fk_reports_recipients_report FOREIGN KEY (report_id) REFERENCES reports (id)
);

-- Запросы для уведомлений
CREATE TABLE queries (
    id SERIAL PRIMARY KEY,
    card_uuid TEXT NOT NULL,
    title TEXT NOT NULL
);

CREATE TABLE templates (
    id SERIAL PRIMARY KEY,
    template_text TEXT, -- text or base64
    title TEXT,
    type text NOT NULL
);

CREATE TABLE crons(
    id SERIAL PRIMARY KEY,
    cron TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_active bool NOT NULL DEFAULT FALSE
);

CREATE TABLE report_crons(
    report_id INT NOT NULL,
    cron_id INT NOT NULL,
    PRIMARY KEY (report_id, cron_id),
    CONSTRAINT fk_report_crons_report FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE,
    CONSTRAINT fk_report_crons_cron FOREIGN KEY (cron_id) REFERENCES crons(id) ON DELETE CASCADE
);


create table export_formats(
    id serial PRIMARY KEY,
    format text
);

create table reports_export(
    report_id int,
    format_id int,
    file_name text,
    order jsonb,
    CONSTRAINT fk_report_export_report FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE,
    CONSTRAINT fk_report_export_export FOREIGN KEY (format_id) REFERENCES export_formats(id) ON DELETE CASCADE
);


create table report_templates(
    report_id INT NOT NULL,
    template_id INT NOT NULL, 
    CONSTRAINT fk_report_template_report FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE,
    CONSTRAINT fk_report_template_template FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE

);

-- Связь многие-ко-многим: reports <-> queries
CREATE TABLE report_queries (
    report_id INT NOT NULL,
    query_id INT NOT NULL,
    PRIMARY KEY (report_id, query_id),
    CONSTRAINT fk_notify_queries_notify FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE,
    CONSTRAINT fk_notify_queries_query FOREIGN KEY (query_id) REFERENCES queries(id) ON DELETE CASCADE
);


-- Триггер: при удалении чата уведомления становятся неактивными
create or replace function deactivate_notify_on_chat_delete()
returns trigger
as $$
BEGIN
    UPDATE reports SET active = FALSE WHERE chat_id = OLD.id;
    RETURN OLD;
END;
$$
language plpgsql
;

CREATE TRIGGER trg_deactivate_notify_on_chat_delete
BEFORE DELETE ON chats
FOR EACH ROW
EXECUTE FUNCTION deactivate_notify_on_chat_delete();

