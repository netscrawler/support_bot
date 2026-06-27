package smtp

import (
	"bytes"
)

type Mail struct {
	Recipients []string `json:"recipients"`
	Copy       []string `json:"copy"`

	Subject string `json:"subject"`

	Body string `json:"body"`

	Attachments []Attachment
}

type Attachment struct {
	File *bytes.Buffer
	Name string
}
