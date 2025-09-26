BEGIN;

-- 1. Создать новую таблицу для связей многие-ко-многим
CREATE TABLE notify_queries (
    notify_id INT NOT NULL,
    query_id INT NOT NULL,
    PRIMARY KEY (notify_id, query_id),
    CONSTRAINT fk_notify_queries_notify FOREIGN KEY (notify_id) REFERENCES notify(id) ON DELETE CASCADE,
    CONSTRAINT fk_notify_queries_query FOREIGN KEY (query_id) REFERENCES queries(id) ON DELETE CASCADE
);

-- 2. Переносим существующие данные из notify.query_id
INSERT INTO notify_queries (notify_id, query_id)
SELECT id, query_id
FROM notify
WHERE query_id IS NOT NULL;

-- 3. Удаляем внешний ключ и колонку query_id
ALTER TABLE notify DROP CONSTRAINT fk_notify_query;
ALTER TABLE notify DROP COLUMN query_id;

COMMIT;

