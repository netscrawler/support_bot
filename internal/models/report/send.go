package models

import (
	"bytes"
	"iter"
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

type SendKind int

const (
	SendTextKind SendKind = iota
	SendImageKind
	SendFileKind
)

type ReportData interface {
	Kind() SendKind
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

type ImageData struct {
	img   []*bytes.Buffer
	name  []string
	Entry int
}

func NewImageData(img *bytes.Buffer, name string) (*ImageData, error) {
	f := []*bytes.Buffer{}
	n := []string{}

	pn, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return nil, err
	}

	return &ImageData{
		img:   append(f, img),
		name:  append(n, pn),
		Entry: len(f),
	}, nil
}

func NewEmptyImageData() *ImageData {
	f := []*bytes.Buffer{}
	n := []string{}

	return &ImageData{
		img:   f,
		name:  n,
		Entry: 0,
	}
}

func (id *ImageData) Extend(img *bytes.Buffer, name string) error {
	n, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return err
	}

	id.img = append(id.img, img)
	id.name = append(id.name, n)
	id.Entry++

	return nil
}

func (id *ImageData) Data() iter.Seq2[*bytes.Buffer, string] {
	return func(yield func(*bytes.Buffer, string) bool) {
		for i := 0; i < len(id.img) && i < len(id.name); i++ {
			if !yield(id.img[i], id.name[i]) {
				return
			}
		}
	}
}

func (*ImageData) Kind() SendKind { return SendImageKind }

type FileData struct {
	file  []*bytes.Buffer
	name  []string
	Entry int
}

func NewFileData(file *bytes.Buffer, name string) (*FileData, error) {
	f := []*bytes.Buffer{}
	n := []string{}

	nm, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return nil, err
	}

	return &FileData{
		file:  append(f, file),
		name:  append(n, nm),
		Entry: 1,
	}, nil
}

func NewEmptyFileData() *FileData {
	f := []*bytes.Buffer{}
	n := []string{}

	return &FileData{
		file:  f,
		name:  n,
		Entry: 0,
	}
}

func (fd *FileData) Len() int {
	return len(fd.file)
}

func (fd *FileData) Extend(file *bytes.Buffer, name string) error {
	nm, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return err
	}

	fd.file = append(fd.file, file)
	fd.name = append(fd.name, nm)
	fd.Entry++

	return nil
}

func (fd *FileData) ExtendWithoutTemplate(file *bytes.Buffer, name string) {
	fd.file = append(fd.file, file)
	fd.name = append(fd.name, name)
	fd.Entry++
}

func (fd *FileData) Append(datas ...FileData) {
	for _, d := range datas {
		fd.file = append(fd.file, d.file...)
		fd.name = append(fd.name, d.name...)
		fd.Entry += d.Entry
	}
}

func (*FileData) Kind() SendKind { return SendFileKind }

func (fd *FileData) Data() iter.Seq2[*bytes.Buffer, string] {
	return func(yield func(*bytes.Buffer, string) bool) {
		for i := 0; i < len(fd.file) && i < len(fd.name); i++ {
			if !yield(fd.file[i], fd.name[i]) {
				return
			}
		}
	}
}
