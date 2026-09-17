package service

import "context"

// ScriptSaver — узкий интерфейс сохранения Lua-скрипта, реализуемый
// store.ScriptStore. Выделен отдельно, чтобы ScriptManager не зависел от
// полного набора методов ScriptStore и был тестируем через мок.
type ScriptSaver interface {
	Save(ctx context.Context, name, script string) error
}

type ScriptManager struct {
	store ScriptSaver
}

func NewScriptManager(store ScriptSaver) *ScriptManager {
	return &ScriptManager{store: store}
}

func (m *ScriptManager) Save(ctx context.Context, name, script string) error {
	return m.store.Save(ctx, name, script)
}
