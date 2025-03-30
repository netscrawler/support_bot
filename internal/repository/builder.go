package repository

import (
	"support_bot/internal/database/postgres"

	"go.uber.org/zap"
)

type RepositoryBuilder struct {
	log *zap.Logger
	s   *postgres.Storage
}

func NewRB(log *zap.Logger, s *postgres.Storage) *RepositoryBuilder {
	return &RepositoryBuilder{
		log: log,
		s:   s,
	}
}

func (rb *RepositoryBuilder) Build() (*User, *Chat) {
	uRepo := NewUser(rb.s, rb.log)
	cRepo := NewChat(rb.s, rb.log)
	return &uRepo, &cRepo
}
