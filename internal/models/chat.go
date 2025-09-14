package models

// Chat представляет чат для отправки уведомлений.
type Chat struct {
	ID          int    `json:"id"`
	ChatID      int64  `json:"chat_id"`
	Title       string `json:"title"`
	Type        string `json:"type"` // 'private', 'group', 'supergroup', 'channel'
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	ThreadID    int64
}

func NewChat(id int64, title, cType, desc string) *Chat {
	return &Chat{
		ChatID:      id,
		Title:       title,
		Type:        cType,
		Description: desc,
	}
}

func (c *Chat) Activate() {
	c.IsActive = true
}

func (c *Chat) DeActivate() {
	c.IsActive = false
}
