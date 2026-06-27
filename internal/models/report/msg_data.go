package models

import (
	"bytes"
	"support_bot/internal/pkg/text"
)

type SendKind int

const (
	SendTextKind SendKind = iota
	SendImageKind
	SendFileKind
)

type Data struct {
	Data *bytes.Buffer
	Name string

	Type SendKind
}

func NewTextData(text *bytes.Buffer) Data {
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
		Data: image,
		Name: n,
		Type: SendImageKind,
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
		Type: SendFileKind,
	}, nil
}

func (d Data) Kind() SendKind { return d.Type }
