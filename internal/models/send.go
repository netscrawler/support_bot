package models

type SenderProvider struct {
	tg   TgSender
	smb  SmbSender
	smtp SmtpSender
	max  MaxSender
}

func NewSenderProvider(tg TgSender, smb SmbSender, smtp SmtpSender, max MaxSender) *SenderProvider {
	return &SenderProvider{tg, smb, smtp, max}
}

func (s SenderProvider) Tg() TgSender {
	return s.tg
}

func (s SenderProvider) SMB() SmbSender {
	return s.smb
}

func (s SenderProvider) SMTP() SmtpSender {
	return s.smtp
}

func (s SenderProvider) MAX() MaxSender {
	return s.max
}
