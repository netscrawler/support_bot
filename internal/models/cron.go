package models

import (
	"errors"
	"fmt"

	"github.com/robfig/cron/v3"
)

type SheduleUnit struct {
	Crontab string `db:"cron"`
	Name    string `db:"name"`

	EventType int `db:"event_type"`
}

// ReportCronEvent — DTO, которое возвращает internal/store.CronStore.
// Живёт в models, а не в internal/event_creator (единственном потребителе),
// чтобы internal/store не зависел от конкретного пакета-потребителя —
// такая зависимость была бы нарушением слоёв (storage не должен знать
// о event_creator).
type ReportCronEvent struct {
	Name     string
	CronName string
}

func (s SheduleUnit) String() string {
	return fmt.Sprintf("%s: %s", s.Name, s.Crontab)
}

type CronVO string

var ErrInvalidCron = errors.New("invalid cron")

func NewCron(cronExpr string) (CronVO, error) {
	_, err := cron.ParseStandard(cronExpr) // 5 полей (без секунд)
	if err != nil {
		return "", ErrInvalidCron
	}

	return CronVO(cronExpr), nil
}
