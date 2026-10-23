package handlers

import (
	"strings"
	"support_bot/internal/models"
	"testing"
	"time"
)

func TestFormatRun(t *testing.T) {
	t0 := time.Date(2026, 1, 2, 15, 4, 0, 0, time.Local)
	run := func(st models.RunStatus, reason, errText string) models.Run {
		return models.Run{
			Kind: models.RunKindCron, Status: st, Reason: reason, Error: errText,
			StartedAt: t0, FinishedAt: t0.Add(3240 * time.Millisecond),
		}
	}

	tests := []struct {
		name string
		in   models.Run
		want string
	}{
		{"success", run(models.RunStatusSuccess, "", ""), "02.01 15:04 · cron · ✅ success · 3.2s"},
		{
			"long error", run(models.RunStatusFailed, "", strings.Repeat("ы", 300)),
			"02.01 15:04 · cron · ❌ failed · 3.2s · " + strings.Repeat("ы", 200) + "…",
		},
		{
			"escape", run(models.RunStatusFailed, "", "a<b"),
			"02.01 15:04 · cron · ❌ failed · 3.2s · a&lt;b",
		},
		{
			"reason", run(models.RunStatusSkipped, "evaluate", ""),
			"02.01 15:04 · cron · ⏭ skipped · 3.2s · evaluate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRun(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatRuns_Empty(t *testing.T) {
	got := formatRuns("a<b", nil, nil)
	if !strings.Contains(got, "a&lt;b") || !strings.HasSuffix(got, "Запусков пока нет.") {
		t.Errorf("unexpected: %q", got)
	}
}
