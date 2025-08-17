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


