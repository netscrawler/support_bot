package delivery

import (
	"context"
	"errors"
	"log/slog"

	"support_bot/internal/delivery/smtp"
	models "support_bot/internal/models/report"
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
	Upload(remote string, file *models.FileData, image *models.ImageData) error
}

type SMTPSender interface {
	Send(ctx context.Context, mail smtp.Mail) error
}

type SenderStrategy struct {
	tg   telegramChatSender
	smb  fileUploader
	smtp SMTPSender
	log  *slog.Logger
}

func NewSender(
	tg telegramChatSender,
	smb fileUploader,
	smtp SMTPSender,
	log *slog.Logger,
) *SenderStrategy {
	l := log.With("module", "delivery")

	return &SenderStrategy{
		tg:   tg,
		smb:  smb,
		smtp: smtp,
		log:  l,
	}
}

func (ss *SenderStrategy) Send(
	ctx context.Context,
	metas []models.Targeted,
	data []models.ReportData,
) error {
	var sendError error

	for _, meta := range metas {
		switch meta.Kind() {
		case models.TargetTelegramChatKind:
			chat, ok := meta.(models.TargetTelegramChat)
			if !ok {
				ss.log.ErrorContext(ctx, "invalid target telegram", slog.Any("dest", meta))

				continue
			}

			err := ss.sendTelegramDataStrategy(chat, data)
			if err != nil {
				sendError = errors.Join(sendError, err)
			}
		case models.TargetFileServerKind:
			remote, ok := meta.(models.TargetFileServer)
			if !ok {
				ss.log.ErrorContext(ctx, "invalid target file server", slog.Any("dest", meta))

				continue
			}

			err := ss.sendSMBDataStrategy(remote, data)
			if err != nil {
				sendError = errors.Join(sendError, err)
			}
		case models.TargetEmailKind:
			email, ok := meta.(models.TargetEmail)
			if !ok {
				ss.log.ErrorContext(ctx, "invalid target email recipient", slog.Any("dest", meta))

				continue
			}
			err := ss.sendSMTPDataStrategy(ctx, email, data)
			if err != nil {
				sendError = errors.Join(sendError, err)
			}

		default:
			ss.log.ErrorContext(ctx, "not supported target kind", slog.Any("dest", meta))

			continue
		}
	}

	return nil
}

func (ss *SenderStrategy) sendTelegramDataStrategy(
	target models.TargetTelegramChat,
	datas []models.ReportData,
) error {
	if len(datas) == 0 {
		return errors.New("NOTHING TO SEND")
	}

	for _, data := range datas {
		if data == nil {
			ss.log.Error("empty send data", slog.Any("data", data))
		}

		switch data.Kind() {
		case models.SendTextKind:
			dt, ok := data.(*models.TextData)
			if !ok {
				ss.log.Error("invalid telegram send data", slog.Any("data", data))

				continue
			}

			err := ss.tg.Send(target, *dt)
			if err != nil {
				ss.log.Error("sending error", slog.Any("error", err))
			}
		case models.SendFileKind:
			dt, ok := data.(*models.FileData)
			if !ok {
				ss.log.Error("invalid telegram send data", slog.Any("data", data))

				continue
			}

			err := ss.tg.SendDocument(target, *dt)
			if err != nil {
				ss.log.Error("sending error", slog.Any("error", err))
			}
		case models.SendImageKind:
			dt, ok := data.(*models.ImageData)
			if !ok {
				ss.log.Error("invalid telegram send data", slog.Any("data", data))

				continue
			}

			err := ss.tg.SendMedia(target, *dt)
			if err != nil {
				ss.log.Error("sending error", slog.Any("error", err))
			}
		default:
			ss.log.Error("not supported telegram data", slog.Any("data", data))

			continue
		}
	}

	return nil
}

func (ss *SenderStrategy) sendSMBDataStrategy(
	target models.TargetFileServer,
	datas []models.ReportData,
) error {
	var sendErr error

	for _, data := range datas {
		if data == nil {
			return errors.New("NOTHING TO SEND")
		}

		switch data.Kind() {
		case models.SendFileKind:
			dt, ok := data.(*models.FileData)
			if !ok {
				sendErr = errors.Join(sendErr, errors.New("INVALID SMB SEND DATA"))
			}

			sendErr = errors.Join(sendErr, ss.smb.Upload(target.Dest, dt, nil))
		case models.SendImageKind:
			it, ok := data.(*models.ImageData)
			if !ok {
				sendErr = errors.Join(sendErr, errors.New("INVALID SMB SEND DATA"))
			}

			sendErr = errors.Join(sendErr, ss.smb.Upload(target.Dest, nil, it))
		default:
			sendErr = errors.Join(sendErr, errors.New("NOT SUPPORTED SAMBA DATA TYPE"))
		}
	}

	return sendErr
}

func (ss *SenderStrategy) sendSMTPDataStrategy(
	ctx context.Context,
	target models.TargetEmail,
	datas []models.ReportData,
) error {
	var sendErr error

	sendData := models.NewEmptyFileData()

	for _, data := range datas {
		if data == nil {
			return errors.New("NOTHING TO SEND")
		}

		switch data.Kind() {
		case models.SendFileKind:
			dt, ok := data.(*models.FileData)
			if !ok {
				sendErr = errors.Join(sendErr, errors.New("INVALID SMTP SEND DATA"))
			}

			sendData.Append(*dt)

		case models.SendImageKind:
			it, ok := data.(*models.ImageData)
			if !ok {
				sendErr = errors.Join(sendErr, errors.New("INVALID SMB SEND DATA"))
			}

			for f, n := range it.Data() {
				sendData.ExtendWithoutTemplate(f, n)
			}

		default:
			sendErr = errors.Join(sendErr, errors.New("NOT SUPPORTED SMTP DATA TYPE"))
		}
	}

	mail := smtp.Mail{
		Recipients:  target.Dest,
		Copy:        target.Copy,
		Subject:     target.Subject,
		Body:        target.Body,
		Attachments: *sendData,
	}

	err := ss.smtp.Send(ctx, mail)
	if err != nil {
		sendErr = errors.Join(sendErr, err)
	}

	return sendErr
}
