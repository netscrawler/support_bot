package models

// Chat представляет чат для отправки уведомлений.
type Chat struct {
	ID          int     `json:"id"          db:"id"`
	ChatID      int64   `json:"chat_id"     db:"chat_id"`
	Title       string  `json:"title"       db:"title"`
	Type        string  `json:"type"        db:"type"` // 'private', 'group', 'supergroup', 'channel'
	Description *string `json:"description" db:"description"`
	IsActive    bool    `json:"is_active"   db:"is_active"`
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
