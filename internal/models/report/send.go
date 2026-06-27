package models

type SenderProvider struct {
	tg         tgSender
	smb        smbSender
	smtpSender smtpSender
}

func NewSenderProvider(tg tgSender, smb smbSender, smtpSender smtpSender) *SenderProvider {
	return &SenderProvider{tg, smb, smtpSender}
}

func (s SenderProvider) Tg() tgSender {
	return s.tg
}

func (s SenderProvider) SMB() smbSender {
	return s.smb
}

func (s SenderProvider) SMTP() smtpSender {
	return s.smtpSender
}
