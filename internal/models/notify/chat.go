package models

// Chat представляет чат для отправки уведомлений.
type Chat struct {
	ID          int     `db:"id"          json:"id"`
	ChatID      int64   `db:"chat_id"     json:"chat_id"`
	Title       string  `db:"title"       json:"title"`
	Type        string  `db:"type"        json:"type"` // 'private', 'group', 'supergroup', 'channel'
	Description *string `db:"description" json:"description"`
	IsActive    bool    `db:"is_active"   json:"is_active"`
	ThreadID    int64
}

func NewChat(id int64, title, cType, desc string) *Chat {
	return &Chat{
		ChatID:      id,
		Title:       title,
		Type:        cType,
		Description: &desc,
	}
}

func (c *Chat) Activate() {
	c.IsActive = true
}

func (c *Chat) DeActivate() {
	c.IsActive = false
}
