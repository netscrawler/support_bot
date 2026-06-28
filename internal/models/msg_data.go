package models

import (
	"bytes"

	"support_bot/internal/pkg/text"
)

type sendKind int

const (
	sendTextKind sendKind = iota
	sendImageKind
	sendFileKind
)

type Data struct {
	Data *bytes.Buffer
	Name string

	Type sendKind
}

func NewTextData(text *bytes.Buffer) Data {
	return Data{
		Data: text,
		Type: sendTextKind,
	}
}

func NewImageData(image *bytes.Buffer, name string) (Data, error) {
	n, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return Data{}, err
	}

	return Data{
		Data: image,
		Name: n,
		Type: sendImageKind,
	}, nil
}

func NewFileData(file *bytes.Buffer, name string) (Data, error) {
	n, err := text.ExecuteTemplate(name, nil)
	if err != nil {
		return Data{}, err
	}

	return Data{
		Data: file,
		Name: n,
		Type: sendFileKind,
	}, nil
}

func (d Data) kind() sendKind { return d.Type }
