package models

import (
	"errors"

	"github.com/robfig/cron/v3"
)

type SheduleUnit struct {
	Crontab string `db:"cron"`
	Name    string `db:"name"`
}

type CronVO string

type Cron struct {
	Cron        CronVO
	Name        string
	Description *string
}

var ErrInvalidCron = errors.New("invalid cron")

func NewCron(cronExpr string) (CronVO, error) {
	_, err := cron.ParseStandard(cronExpr) // 5 полей (без секунд)
	if err != nil {
		return "", ErrInvalidCron
	}

	return CronVO(cronExpr), nil
}

type NotificationResult struct {
	Title string
	Text  *string
	HTML  *any
	Image *ImageData
	XLSX  *map[string][][]string
	CSV   *FileData
}
