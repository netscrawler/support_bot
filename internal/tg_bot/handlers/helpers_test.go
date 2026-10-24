package handlers

import (
	"strings"
	"testing"
	"time"

	"support_bot/internal/models"

	"github.com/mymmrac/telego"
)

func TestFormatRun(t *testing.T) {
	t0 := time.Date(2026, 1, 2, 15, 4, 0, 0, time.Local)
	run := func(st models.RunStatus, reason, errText string) models.Run {
		return models.Run{
			Kind:       models.RunKindCron,
			Status:     st,
			Reason:     reason,
			Error:      errText,
			StartedAt:  t0,
			FinishedAt: t0.Add(3240 * time.Millisecond),
		}
	}

	tests := []struct {
		name string
		in   models.Run
		want string
	}{
		{
			"success",
			run(models.RunStatusSuccess, "", ""),
			"02.01 15:04 · cron · ✅ success · 3.2s",
		},
		{
			"long error",
			run(models.RunStatusFailed, "", strings.Repeat("ы", 300)),
			"02.01 15:04 · cron · ❌ failed · 3.2s · " + strings.Repeat(
				"ы",
				200,
			) + "…",
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
	if !strings.Contains(got, "a&lt;b") ||
		!strings.HasSuffix(got, "Запусков пока нет.") {
		t.Errorf("unexpected: %q", got)
	}
}

func TestGetReportInfoFromQueryPeriod(t *testing.T) {
	tests := []struct {
		data    string
		code    string
		wantErr bool
	}{
		{"report_get;1;rep;2", "", false},
		{"report_get;1;rep;2;y", "y", false},
		{"report_get;1;rep;2;y;x", "", true},
		{"report_get;1;rep", "", true},
	}

	for _, tt := range tests {
		got, err := getReportInfoFromQuery(telego.CallbackQuery{Data: tt.data})
		if (err != nil) != tt.wantErr || got.PeriodCode != tt.code {
			t.Errorf("%q: code=%q err=%v", tt.data, got.PeriodCode, err)
		}
	}
}

func TestPeriodStep(t *testing.T) {
	rep := reportInfoFromQuery{
		Callback:   "report_get",
		ID:         "1",
		ReportName: "rep",
		PageFrom:   2,
	}

	t.Run("uses period, no code: keyboard", func(t *testing.T) {
		p, mark := periodStep(rep, true, "back")
		if p != "" || mark == nil {
			t.Fatalf("p=%q mark=%v", p, mark)
		}

		if got := mark.InlineKeyboard[0][1].CallbackData; got != "report_get;1;rep;2;y" {
			t.Errorf("callback = %q", got)
		}

		if got := mark.InlineKeyboard[2][0].CallbackData; got != "back" {
			t.Errorf("back = %q", got)
		}
	})

	t.Run("no period in report: run at once", func(t *testing.T) {
		if p, mark := periodStep(rep, false, "back"); p != "" || mark != nil {
			t.Fatalf("p=%q mark=%v", p, mark)
		}
	})

	t.Run("code y: yesterday", func(t *testing.T) {
		r := rep

		r.PeriodCode = "y"

		if p, mark := periodStep(
			r,
			true,
			"back",
		); p != models.PeriodYesterday ||
			mark != nil {
			t.Fatalf("p=%q mark=%v", p, mark)
		}
	})

	t.Run("unknown code: default period", func(t *testing.T) {
		r := rep

		r.PeriodCode = "zz"

		if p, mark := periodStep(r, true, "back"); p != "" || mark != nil {
			t.Fatalf("p=%q mark=%v", p, mark)
		}
	})

	t.Run("too long name: default period, no keyboard", func(t *testing.T) {
		r := rep

		r.ReportName = strings.Repeat("a", 60)

		if p, mark := periodStep(r, true, "back"); p != "" || mark != nil {
			t.Fatalf("p=%q mark=%v", p, mark)
		}
	})
}
