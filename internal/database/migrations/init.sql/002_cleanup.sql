-- Функция для очистки устаревших уведомлений (старше 7 дней)
CREATE OR REPLACE FUNCTION cleanup_old_notifications()
RETURNS integer AS $$
DECLARE
    deleted_count integer;
BEGIN
    DELETE FROM chat_notifications
    WHERE status = 'sent'
      AND sent_at < NOW() - INTERVAL '7 days';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_old_notifications() IS 'Автоматически удаляет записи об отправленных уведомлениях, которые старше 7 дней';


CREATE EXTENSION IF NOT EXISTS pg_cron;

SELECT cron.schedule('cleanup-notifications', '0 3 * * *', 'SELECT cleanup_old_notifications()');

SELECT * FROM cron.job;
