package service

import (
	"support_bot/internal/adaptors/telegram"
	"support_bot/internal/repository"

	"go.uber.org/zap"
)

type repoBuilder interface {
	Build() (*repository.User, *repository.Chat)
}

type adaptorBuilder interface {
	Build() *telegram.ChatAdaptor
}

// Билдер для создания сервисов
type ServiceBuilder struct {
	log *zap.Logger
	rb  repoBuilder
	ab  adaptorBuilder
}

// NewSB Возвращает новый инстанс билдера
func NewSB(log *zap.Logger, rb repoBuilder, ab adaptorBuilder) *ServiceBuilder {
	return &ServiceBuilder{
		log: log,
		rb:  rb,
		ab:  ab,
	}
}

// Build собирает и возвращает сервисы
func (sb *ServiceBuilder) Build() (*User, *Chat, *ChatNotify, *UserNotify) {
	uRepo, cRepo := sb.rb.Build()
	tgAdaptor := sb.ab.Build()
	uService := newUser(uRepo, sb.log)
	cService := newChat(cRepo, sb.log)
	nService := newChatNotify(cRepo, sb.log, tgAdaptor)
	nuService := newUserNotify(uRepo, sb.log, tgAdaptor)
	return uService, cService, nService, nuService
}
