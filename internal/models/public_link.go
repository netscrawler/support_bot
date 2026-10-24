package models

import "time"

// PublicLink — публичная ссылка на отчёт. ExpiresAt == nil — бессрочная.
type PublicLink struct {
	ID        string
	Report    string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

func (l PublicLink) Active(now time.Time) bool {
	return l.ExpiresAt == nil || l.ExpiresAt.After(now)
}
