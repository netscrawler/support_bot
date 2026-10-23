package models

import "time"

type RunKind string

const (
	RunKindCron   RunKind = "cron"
	RunKindManual RunKind = "manual"
	RunKindLK     RunKind = "lk"
	RunKindTest   RunKind = "test"
)

type RunStatus string

const (
	RunStatusSuccess RunStatus = "success"
	RunStatusSkipped RunStatus = "skipped"
	RunStatusFailed  RunStatus = "failed"
	RunStatusPartial RunStatus = "partial"
)

// Run — запись журнала запусков отчётов. Пустые Reason/Error хранятся в БД как NULL.
type Run struct {
	ReportName string
	Kind       RunKind
	Status     RunStatus
	Reason     string
	Error      string
	ChatID     *int64
	StartedAt  time.Time
	FinishedAt time.Time
}
