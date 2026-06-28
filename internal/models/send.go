package models

type SenderProvider struct {
	tg         TgSender
	smb        SmbSender
	smtpSender SmtpSender
}

func NewSenderProvider(tg TgSender, smb SmbSender, smtpSender SmtpSender) *SenderProvider {
	return &SenderProvider{tg, smb, smtpSender}
}

func (s SenderProvider) Tg() TgSender {
	return s.tg
}

func (s SenderProvider) SMB() SmbSender {
	return s.smb
}

func (s SenderProvider) SMTP() SmtpSender {
	return s.smtpSender
}
