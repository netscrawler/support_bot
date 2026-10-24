package tgbot

import (
	"slices"
	"strings"
	"time"

	"support_bot/internal/metrics"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

const (
	labelOther = "other"
)

// updateLabels maps an update to bounded (type, command) metric labels.
func updateLabels(u telego.Update) (typ, command string) {
	switch {
	case u.Message != nil:
		cmd, ok := strings.CutPrefix(u.Message.Text, "/")
		if !ok {
			return "message", labelOther
		}

		if i := strings.IndexAny(cmd, " @"); i >= 0 {
			cmd = cmd[:i]
		}

		if !slices.Contains(botCommands, cmd) {
			cmd = labelOther
		}

		return "message", cmd

	case u.CallbackQuery != nil:
		for _, c := range botCallbacks {
			if strings.HasPrefix(u.CallbackQuery.Data, c) {
				return "callback", c
			}
		}

		return "callback", labelOther
	}

	return labelOther, labelOther
}

func metricsMiddleware(ctx *th.Context, u telego.Update) error {
	start := time.Now()
	err := ctx.Next(u)

	status := "ok"
	if err != nil {
		status = "error"
	}

	typ, command := updateLabels(u)
	metrics.BotUpdates.WithLabelValues(typ, command, status).Inc()
	metrics.BotUpdateDuration.WithLabelValues(typ).
		Observe(time.Since(start).Seconds())

	return err
}
