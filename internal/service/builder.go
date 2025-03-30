package service

import (
	"support_bot/internal/repository"

	"go.uber.org/zap"
)

type repoBuilder interface {
	Build() (*repository.User, *repository.Chat)
}

type ServiceBuilder struct {
	log *zap.Logger
	rb  repoBuilder
}

func NewSB(log *zap.Logger, rb repoBuilder) *ServiceBuilder {
	return &ServiceBuilder{
		log: log,
		rb:  rb,
	}
}

func (sb *ServiceBuilder) Build() (*User, *Chat, *Notify) {
	uRepo, cRepo := sb.rb.Build()
	uService := newUser(uRepo, sb.log)
	cService := newChat(cRepo, sb.log)
	nService := newNotify(cRepo, sb.log)
	return uService, cService, nService
}
