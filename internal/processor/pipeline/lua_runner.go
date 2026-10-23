package pipeline

import (
	"context"
	"errors"

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
) (models.Dataset, error) {
	var (
		plug *lua.Plugin
		err  error
	)

	switch {
	case step.Script != "":
		plug, err = l.mng.NewPlugin(step.Script)
	case step.ScriptName != "":
		plug, err = l.mng.NewPluginFromName(ctx, step.ScriptName)
	default:
		return nil, errors.New(
			"unknown lua plugin from step, step must contain script_name or script",
		)
	}

	if err != nil {
		return nil, err
	}

	out, err := plug.Execute(ctx, data)
	if err != nil {
		return nil, err
	}

	return out, nil
}
