package models

type SenderProvider struct {
	tg   TgSender
	smb  SmbSender
	smtp SMTPSender
	max  MaxSender
}

func NewSenderProvider(
	tg TgSender,
	smb SmbSender,
	smtp SMTPSender,
	maxSender MaxSender,
) *SenderProvider {
	return &SenderProvider{tg, smb, smtp, maxSender}
}

func (s SenderProvider) Tg() TgSender {
	return s.tg
}

func (s SenderProvider) SMB() SmbSender {
	return s.smb
}

func (s SenderProvider) SMTP() SMTPSender {
	return s.smtp
}

func (s SenderProvider) MAX() MaxSender {
	return s.max
}
