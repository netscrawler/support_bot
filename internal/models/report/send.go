package models

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"support_bot/internal/pkg/text"
)

type TargetKind int

const (
	TargetTelegramChatKind TargetKind = iota
	TargetFileServerKind
	TargetEmailKind
)

// TelegramParseMode determines the way client applications treat the text of the message.
type TelegramParseMode = string

const (
	TelegramParseModeDefault    TelegramParseMode = ""
	TelegramParseModeMarkdown   TelegramParseMode = "Markdown"
	TelegramParseModeMarkdownV2 TelegramParseMode = "MarkdownV2"
	TelegramParseModeHTML       TelegramParseMode = "HTML"
)

type Targeted interface {
	Kind() TargetKind
}

type TargetTelegramChat struct {
	ChatID   int64
	ThreadID int
}

func NewTargetTelegramChat(chat int64, thread *int) TargetTelegramChat {
	t := 0

	if thread != nil {
		t = *thread
	}

	return TargetTelegramChat{ChatID: chat, ThreadID: t}
}

func (t TargetTelegramChat) Kind() TargetKind { return TargetTelegramChatKind }

type TargetFileServer struct {
	Dest string
}

func (t TargetFileServer) Kind() TargetKind { return TargetFileServerKind }

type TargetEmail struct {
	Dest    []string
	Copy    []string
	Body    string
	Subject string
}

func NewTargetEmail(tmpl EmailTemplate) (TargetEmail, error) {
	subject, err := text.ExecuteTemplate(tmpl.Subject, nil)
	if err != nil {
		return TargetEmail{}, err
	}

	var body string

	if tmpl.Body != nil {
		body, err = text.ExecuteTemplate(*tmpl.Body, nil)
		if err != nil {
			return TargetEmail{}, err
		}
	} else {
		body = ""
	}

	return TargetEmail{
		Dest:    tmpl.Dest,
		Copy:    tmpl.Copy,
		Body:    body,
		Subject: subject,
	}, nil
}

func (t TargetEmail) Kind() TargetKind { return TargetEmailKind }

type SendKind string

const (
	SendTextKind  SendKind = "text"
	SendImageKind          = "image"
	SendFileKind           = "File"
)

func (s SendKind) String() string {
	return string(s)
}

type ExportedReport struct {
	Raw    string       `json:"raw,omitempty"`
	Type   string       `json:"type,omitempty"`
	Config ReportConfig `json:"config,omitempty"`
}

const (
	TextType  = "text"
	FileType  = "file"
	ImageType = "image"
)

func (er *ExportedReport) ToTextData() (*TextData, error) {
	if er.Type != TextType {
		return nil, fmt.Errorf("expected text, got %s", er.Type)
	}

	txt, err := base64.StdEncoding.DecodeString(er.Raw)
	if err != nil {
		return nil, err
	}

	return NewTextData(string(txt)), nil
}

func (er *ExportedReport) ToFileData() (*FileData, error) {
	if er.Type != FileType {
		return nil, fmt.Errorf("expected file, got %s", er.Type)
	}

	buf, err := base64.StdEncoding.DecodeString(er.Raw)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode exported report: %w", err)
	}

	if er.Config.FileName == nil {
		return nil, fmt.Errorf("empty file name")
	}

	switch er.Config.SendKind {
	case new(FileType):
		return NewFileData(bytes.NewBuffer(buf), *er.Config.FileName)
	case new(SendImageKind):
		return NewImageData(bytes.NewBuffer(buf), *er.Config.FileName)
	case nil:
		fallthrough
	default:
		return nil, fmt.Errorf("unexpected send kind")

	}
}

type ReportConfig struct {
	FileName  *string `json:"file_name,omitempty"`
	ParseMode *string `json:"parse_mode,omitempty"`
	SendKind  *string `json:"send_kind,omitempty"`
}

type TextData struct {
	Msg   string
	Parse TelegramParseMode
}

func NewTextData(text string) *TextData {
	p := TelegramParseModeHTML

	return &TextData{
		Msg:   text,
		Parse: p,
	}
}

func (TextData) Kind() SendKind { return SendTextKind }

func (t TextData) Export() (*ExportedReport, error) {
	var buf bytes.Buffer
	_, err := base64.NewEncoder(base64.StdEncoding, &buf).Write([]byte(t.Msg))
	if err != nil {
		return nil, err
	}

	return &ExportedReport{
		Raw:  buf.String(),
		Type: TextType,
		Config: ReportConfig{
			ParseMode: &t.Parse,
		},
	}, nil
}

type FileData struct {
	File *bytes.Buffer
	Name string
	kind SendKind
}

func NewFileData(file *bytes.Buffer, name string) (*FileData, error) {
	nm, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return nil, err
	}

	return &FileData{
		File: file,
		Name: nm,
		kind: SendFileKind,
	}, nil
}

func NewImageData(file *bytes.Buffer, name string) (*FileData, error) {
	nm, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return nil, err
	}

	return &FileData{
		File: file,
		Name: nm,
		kind: SendImageKind,
	}, nil
}
func (FileData) Kind() SendKind { return SendFileKind }

func (f FileData) Export() (*ExportedReport, error) {
	var buf bytes.Buffer
	_, err := base64.NewEncoder(base64.StdEncoding, &buf).Write(f.File.Bytes())
	if err != nil {
		return nil, err
	}

	return &ExportedReport{
		Raw:    buf.String(),
		Type:   FileType,
		Config: ReportConfig{SendKind: new(f.kind.String()), FileName: &f.Name},
	}, nil
}
