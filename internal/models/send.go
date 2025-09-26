package models

import (
	"bytes"
	"iter"
)

type TargetKind int

const (
	TargetTelegramChatKind TargetKind = iota
	TargetFileServerKind
	TargetEmailKind
)

// ParseMode determines the way client applications treat the text of the message
type ParseMode = string

const (
	ParseModeDefault    ParseMode = ""
	ParseModeMarkdown   ParseMode = "Markdown"
	ParseModeMarkdownV2 ParseMode = "MarkdownV2"
	ParseModeHTML       ParseMode = "HTML"
)

type Targeted interface {
	Kind() TargetKind
}

type TargetTelegramChat struct {
	ChatID   int64 `yaml:"chat_id"   env:"chat_id"`
	ThreadID int   `yaml:"thread_id" env:"thread_id"`
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
	Dest string
}

func (t TargetEmail) Kind() TargetKind { return TargetEmailKind }

type SendKind int

const (
	SendTextKind SendKind = iota
	SendImageKind
	SendFileKind
)

type Sendable interface {
	Kind() SendKind
}

type TextData struct {
	Msg   string
	Parse ParseMode
}

func NewTextData(text string, parse *ParseMode) TextData {
	p := ParseModeMarkdownV2
	if parse != nil {
		p = *parse
	}

	return TextData{
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

func NewImageData(img *bytes.Buffer, name string) *ImageData {
	f := []*bytes.Buffer{}
	n := []string{}
	return &ImageData{
		img:   append(f, img),
		name:  append(n, name),
		Entry: len(f),
	}
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

func (id *ImageData) Extend(img *bytes.Buffer, name string) {
	id.img = append(id.img, img)
	id.name = append(id.name, name)
	id.Entry += 1
}

func (id *ImageData) ExtendIter(data iter.Seq2[*bytes.Buffer, string]) {
	for i, n := range data {
		id.img = append(id.img, i)
		id.name = append(id.name, n)
		id.Entry += 1

	}
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

func (ImageData) Kind() SendKind { return SendImageKind }

type FileData struct {
	file  []*bytes.Buffer
	name  []string
	Entry int
}

func NewFileData(file *bytes.Buffer, name string) *FileData {
	f := []*bytes.Buffer{}
	n := []string{}
	return &FileData{
		file:  append(f, file),
		name:  append(n, name),
		Entry: len(f),
	}
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

func (fd *FileData) ExtendIter(data iter.Seq2[*bytes.Buffer, string]) {
	for i, n := range data {
		fd.file = append(fd.file, i)
		fd.name = append(fd.name, n)
		fd.Entry += 1

	}
}

func (fd *FileData) Extend(file *bytes.Buffer, name string) {
	fd.file = append(fd.file, file)
	fd.name = append(fd.name, name)
	fd.Entry += 1
}

func (FileData) Kind() SendKind { return SendFileKind }

func (fd *FileData) Data() iter.Seq2[*bytes.Buffer, string] {
	return func(yield func(*bytes.Buffer, string) bool) {
		for i := 0; i < len(fd.file) && i < len(fd.name); i++ {
			if !yield(fd.file[i], fd.name[i]) {
				return
			}
		}
	}
}
