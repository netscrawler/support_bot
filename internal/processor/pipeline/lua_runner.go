package pipeline

import (
	"context"
	"errors"
	"time"

	"support_bot/internal/metrics"
	"support_bot/internal/models"
	"support_bot/internal/processor/lua"
)

type LuaRunner struct {
	mng *lua.Manager
}

func NewLuaRunner(mng *lua.Manager) *LuaRunner {
	return &LuaRunner{mng: mng}
}

func (l *LuaRunner) Run(
	ctx context.Context,
	step Step,
	data models.Dataset,
) (_ models.Dataset, err error) {
	var (
		plug   *lua.Plugin
		script string
	)

	start := time.Now()

	switch {
	case step.Script != "":
		script = "inline"
		plug, err = l.mng.NewPlugin(step.Script)

	case step.ScriptName != "":
		script = step.ScriptName
		plug, err = l.mng.NewPluginFromName(ctx, step.ScriptName)

	default:
		return nil, errors.New(
			"unknown lua plugin from step, step must contain script_name or script",
		)
	}

	defer func() {
		metrics.LuaDuration.WithLabelValues(script).
			Observe(time.Since(start).Seconds())

		if err != nil {
			metrics.LuaErrors.WithLabelValues(script, metrics.Reason(err)).Inc()
		}
	}()

	if err != nil {
		return nil, err
	}

	out, err := plug.Execute(ctx, data)
	if err != nil {
		return nil, err
	}

	return out, nil
}
