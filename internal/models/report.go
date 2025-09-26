package models

import (
	"errors"

	"github.com/robfig/cron/v3"
)

type Cron string

var ErrInvalidCron = errors.New("invalid cron")

func NewCron(cronExpr string) (Cron, error) {
	_, err := cron.ParseStandard(cronExpr) // 5 полей (без секунд)
	if err != nil {
		return "", ErrInvalidCron
	}
	return Cron(cronExpr), nil
}

type ReportFormat = string

const (
	NotifyFormatText ReportFormat = "text"
	NotifyFormatHTML ReportFormat = "html"
	NotifyFormatPng  ReportFormat = "png"
	NotifyFormatCsv  ReportFormat = "csv"
	NotifyFormatXlsx ReportFormat = "xlsx"
)

var FormatMap = map[string]ReportFormat{
	"text": NotifyFormatText,
	"html": NotifyFormatHTML,
	"png":  NotifyFormatPng,
	"csv":  NotifyFormatCsv,
	"xlsx": NotifyFormatXlsx,
}

type Report struct {
	Name         string
	GroupID      string   // group of notify for grouping many queries
	CardUUID     []string // metabase card uuid
	Cron         Cron     // cron settings
	TemplateText *string  // template for text type notify
	Title        string
	GroupTitle   string
	Target       Targeted
	Active       bool
	NotifyFormat []ReportFormat // format of notification [png | xlsx | text | csv | or many]
}

func NewReport(
	name, groupID string,
	cardUUID []string,
	cronExpr string,
	templateText *string,
	title string,
	groupTitle string,
	chatID int64,
	remotePath *string,
	threadID int,
	targetKind TargetKind,
	active bool,
	reportFormat []string,
) (Report, error) {
	cron, err := NewCron(cronExpr)
	if err != nil {
		return Report{}, err
	}

	nf := make([]ReportFormat, 0, len(reportFormat))

	for _, f := range reportFormat {
		if format, ok := FormatMap[f]; ok {
			nf = append(nf, format)
		}
	}

	var t Targeted

	switch targetKind {
	case TargetTelegramChatKind:
		t = TargetTelegramChat{
			ChatID:   chatID,
			ThreadID: int(threadID),
		}
	case TargetFileServerKind:
		t = TargetFileServer{
			Dest: *remotePath,
		}
	}

	return Report{
		Name:         name,
		GroupID:      groupID,
		CardUUID:     cardUUID,
		Cron:         cron,
		TemplateText: templateText,
		Title:        title,
		Active:       active,
		NotifyFormat: nf,
		GroupTitle:   groupTitle,
		Target:       t,
	}, nil
}

type NotificationResult struct {
	Title string
	Text  *string
	HTML  *any
	Image *ImageData
	XLSX  *map[string][][]string
	CSV   *FileData
}
