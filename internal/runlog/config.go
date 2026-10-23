package runlog

import "time"

type Config struct {
	Retention time.Duration `env:"RUNS_RETENTION" yaml:"retention" comment:"Сколько хранить записи журнала запусков отчётов." env-default:"720h"`
}

type AlertsConfig struct {
	ChatID   int64 `env:"ALERTS_CHAT_ID"   yaml:"chat_id"   comment:"Чат для уведомлений о сбоях отчётов. Если 0 — уведомления уходят всем админам в личку."`
	ThreadID int   `env:"ALERTS_THREAD_ID" yaml:"thread_id" comment:"Тема (thread) в чате уведомлений, необязательно."`
}
