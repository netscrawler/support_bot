package smtp

import models "support_bot/internal/models/report"

type Mail struct {
	Recipients []string `json:"recipients"`
	Copy       []string `json:"copy"`

	Subject string `json:"subject"`

	Body string `json:"body"`

	Attachments models.FileData
}
