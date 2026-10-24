package tgbot

import (
	"errors"
	"testing"
	"time"

	"support_bot/internal/metrics"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func msg(text string) telego.Update {
	return telego.Update{Message: &telego.Message{Text: text}}
}

func cb(data string) telego.Update {
	return telego.Update{CallbackQuery: &telego.CallbackQuery{Data: data}}
}

func TestUpdateLabels(t *testing.T) {
	tests := []struct {
		name         string
		u            telego.Update
		typ, command string
	}{
		{"start", msg("/start"), "message", "start"},
		{"start with bot and args", msg("/start@bot x"), "message", "start"},
		{"unknown command", msg("/foo"), "message", "other"},
		{"plain text", msg("hello"), "message", "other"},
		{"callback prefix", cb("report_gen_today"), "callback", "report_gen"},
		{
			"callback specific first",
			cb("admin_report_page;3"),
			"callback",
			"admin_report_page",
		},
		{"unknown callback", cb("zzz"), "callback", "other"},
		{"other update", telego.Update{}, "other", "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, command := updateLabels(tt.u)
			assert.Equal(t, tt.typ, typ)
			assert.Equal(t, tt.command, command)
		})
	}
}

func TestMetricsMiddleware(t *testing.T) {
	bot, err := telego.NewBot("1234567890:aaaabbbbaaaabbbbaaaabbbbaaaabbbbccc")
	require.NoError(t, err)

	updates := make(chan telego.Update)
	bh, err := th.NewBotHandler(bot, updates)
	require.NoError(t, err)

	base := bh.BaseGroup()
	base.Use(metricsMiddleware)
	base.HandleMessage(
		func(*th.Context, telego.Message) error { return errors.New("boom") },
		th.CommandEqual("start"),
	)

	done := make(chan struct{})
	go func() { _ = bh.Start(); close(done) }()

	require.Eventually(t, bh.IsRunning, time.Second, time.Millisecond)

	errC := metrics.BotUpdates.WithLabelValues("message", "start", "error")
	unmatchedC := metrics.BotUpdates.WithLabelValues("message", "other", "ok")
	errBefore, unmatchedBefore := testutil.ToFloat64(
		errC,
	), testutil.ToFloat64(
		unmatchedC,
	)

	updates <- msg("/start")

	updates <- msg("no handler")

	close(updates)
	<-done
	require.NoError(t, bh.StopWithContext(t.Context()))

	assert.InDelta(t, errBefore+1, testutil.ToFloat64(errC), 0)
	assert.InDelta(
		t,
		unmatchedBefore+1,
		testutil.ToFloat64(unmatchedC),
		0,
		"unmatched update passes base middleware",
	)
}
