package models

import (
	"bytes"

	"support_bot/internal/pkg/text"
)

type sendKind int

const (
	SendTextKind sendKind = iota
	sendImageKind
	sendFileKind
	SendRichTextKind
)

type Data struct {
	Data     *bytes.Buffer
	FileName string

	Type sendKind
}

func (d Data) Read(p []byte) (n int, err error) {
	return d.Data.Read(p)
}

func (d Data) Name() string {
	return d.FileName
}

func NewTextData(text *bytes.Buffer, eType string) Data {
	if eType == "rich_text" {
		return Data{
			Data: text,
			Type: SendRichTextKind,
		}
	}

	return Data{
		Data: text,
		Type: SendTextKind,
	}
}

func NewImageData(image *bytes.Buffer, name string) (Data, error) {
	n, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return Data{}, err
	}

	return Data{
		Data:     image,
		FileName: n,
		Type:     sendImageKind,
	}, nil
}

func NewFileData(file *bytes.Buffer, name string) (Data, error) {
	n, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return Data{}, err
	}

	return Data{
		Data:     file,
		FileName: n,
		Type:     sendFileKind,
	}, nil
}

func (d Data) kind() sendKind { return d.Type }
