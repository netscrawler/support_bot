package runlog

import "time"

type Config struct {
	Retention time.Duration `comment:"Сколько хранить записи журнала запусков отчётов." env:"RUNS_RETENTION" env-default:"720h" yaml:"retention"`
}

type AlertsConfig struct {
	ChatID   int64 `comment:"Чат для уведомлений о сбоях отчётов. Если 0 — уведомления уходят всем админам в личку."                                                                       env:"ALERTS_CHAT_ID"   yaml:"chat_id"`
	ThreadID int   `comment:"Тема (thread) в чате уведомлений, необязательно."                                                                                                             env:"ALERTS_THREAD_ID" yaml:"thread_id"`
}
