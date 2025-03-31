package service

import (
	"support_bot/internal/repository"

	"go.uber.org/zap"
)

type repoBuilder interface {
	Build() (*repository.User, *repository.Chat)
}

// Билдер для создания сервисов
type ServiceBuilder struct {
	log *zap.Logger
	rb  repoBuilder
}

// NewSB Возвращает новый инстанс билдера
func NewSB(log *zap.Logger, rb repoBuilder) *ServiceBuilder {
	return &ServiceBuilder{
		log: log,
		rb:  rb,
	}
}

// Build собирает и возвращает сервисы
func (sb *ServiceBuilder) Build() (*User, *Chat, *ChatNotify, *UserNotify) {
	uRepo, cRepo := sb.rb.Build()
	uService := newUser(uRepo, sb.log)
	cService := newChat(cRepo, sb.log)
	nService := newChatNotify(cRepo, sb.log)
	nuService := newUserNotify(uRepo, sb.log)
	return uService, cService, nService, nuService
}
