package models

import (
	"bytes"
	"errors"
	"text/template"
	"time"

	"github.com/robfig/cron/v3"
)

type Cron string

var ErrInvalidCron = errors.New("invalid cron")

type NotifyFormat = string

const (
	NotifyFormatText NotifyFormat = "text"
	NotifyFormatPng  NotifyFormat = "png"
	NotifyFormatCsv  NotifyFormat = "csv"
	NotifyFormatXlsx NotifyFormat = "xlsx"
)

var FormatMap = map[string]NotifyFormat{
	"text": NotifyFormatText,
	"png":  NotifyFormatPng,
	"csv":  NotifyFormatCsv,
	"xlsx": NotifyFormatXlsx,
}

func NewCron(cronExpr string) (Cron, error) {
	_, err := cron.ParseStandard(cronExpr) // 5 полей (без секунд)
	if err != nil {
		return "", ErrInvalidCron
	}
	return Cron(cronExpr), nil
}

type PngImage struct {
	Data  *bytes.Buffer
	Title string
}

type Notify struct {
	Name         string
	GroupID      string  // group of notify for grouping many queries
	CardUUID     string  // metabase card uuid
	Cron         Cron    // cron settings
	TemplateText *string // template for text type notify
	Title        string
	GroupTitle   string
	ChatID       int64 // telegram chat id
	Active       bool
	NotifyFormat []NotifyFormat // format of notification [png | xlsx | text | csv | or many]
}

func NewNotify(
	name, groupID string,
	cardUUID string,
	cronExpr string,
	templateText *string,
	title string,
	groupTitle string,
	chatID int64,
	active bool,
	notifyFormat []string,
) (Notify, error) {
	cron, err := NewCron(cronExpr)
	if err != nil {
		return Notify{}, err
	}

	nf := make([]NotifyFormat, 0, len(notifyFormat))

	for _, f := range notifyFormat {
		if format, ok := FormatMap[f]; ok {
			nf = append(nf, format)
		}
	}

	return Notify{
		Name:         name,
		GroupID:      groupID,
		CardUUID:     cardUUID,
		Cron:         cron,
		TemplateText: templateText,
		Title:        title,
		ChatID:       chatID,
		Active:       active,
		NotifyFormat: nf,
		GroupTitle:   groupTitle,
	}, nil
}

func (n Notify) GetGroupTitle() (string, error) {
	t := newTitlePlaceholder()
	temp, err := template.New("GroupTitle").Parse(n.GroupTitle)
	if err != nil {
		return n.GroupTitle, err
	}

	var buf bytes.Buffer
	temp.Execute(&buf, t)
	return buf.String(), nil
}

type titlePlaceHolder struct {
	CurrentDate string
	LastDate    string
}

func newTitlePlaceholder() *titlePlaceHolder {
	const format = "02-01-2006"
	t := time.Now()
	return &titlePlaceHolder{
		CurrentDate: t.Format(format),
		LastDate:    t.Add(-time.Hour * 24).Format(format),
	}
}
