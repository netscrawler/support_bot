package service

import (
	"errors"

	"support_bot/internal/models"
)

type telegramChatSender interface {
	Send(chat models.TargetTelegramChat, msg models.TextData) error
	SendDocument(
		chat models.TargetTelegramChat,
		doc models.FileData,
	) error
	SendMedia(chat models.TargetTelegramChat, imgs models.ImageData) error
}

type fileUploader interface {
	Upload(remote string, file *models.FileData) error
}

type SenderStrategy struct {
	tg  telegramChatSender
	smb fileUploader
}

func NewSender(tg telegramChatSender, smb fileUploader) *SenderStrategy {
	return &SenderStrategy{
		tg:  tg,
		smb: smb,
	}
}

func (ss *SenderStrategy) Send(meta models.Targeted, data models.Sendable) error {
	switch meta.Kind() {
	case models.TargetTelegramChatKind:
		chat, ok := meta.(models.TargetTelegramChat)
		if !ok {
			return errors.New("INVALID TARGET TELEGRAM")
		}

		return ss.sendTelegramDataStrategy(chat, data)
	case models.TargetFileServerKind:
		remote, ok := meta.(models.TargetFileServer)
		if !ok {
			return errors.New("INVALID TARGET FILE_SERVER")
		}

		return ss.sendSMBDataStrategy(remote, data)
	case models.TargetEmailKind:
		return nil
	default:
		return errors.New("NOT SUPPORTED TARGET KIND")
	}
}

func (ss *SenderStrategy) sendTelegramDataStrategy(
	target models.TargetTelegramChat,
	data models.Sendable,
) error {
	if data == nil {
		return errors.New("NOTHING TO SEND")
	}

	switch data.Kind() {
	case models.SendTextKind:
		dt, ok := data.(models.TextData)
		if !ok {
			return errors.New("INVALID TELEGRAM SEND DATA")
		}

		return ss.tg.Send(target, dt)
	case models.SendFileKind:
		dt, ok := data.(*models.FileData)
		if !ok {
			return errors.New("INVALID TELEGRAM SEND DATA")
		}

		return ss.tg.SendDocument(target, *dt)
	case models.SendImageKind:
		dt, ok := data.(*models.ImageData)
		if !ok {
			return errors.New("INVALID TELEGRAM SEND DATA")
		}

		return ss.tg.SendMedia(target, *dt)
	default:
		return errors.New("NOT SUPPORTED TELEGRAM DATA TYPE")
	}
}

func (ss *SenderStrategy) sendSMBDataStrategy(
	target models.TargetFileServer,
	data models.Sendable,
) error {
	if data == nil {
		return errors.New("NOTHING TO SEND")
	}

	switch data.Kind() {
	case models.SendFileKind:
		dt, ok := data.(*models.FileData)
		if !ok {
			return errors.New("INVALID TELEGRAM SEND DATA")
		}

		return ss.smb.Upload(target.Dest, dt)
	default:
		return errors.New("NOT SUPPORTED SAMBA DATA TYPE")
	}
}
