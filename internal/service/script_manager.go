package service

import "support_bot/internal/repository"

type ScriptManager struct {
	*repository.Script
}

func NewScriptManager(r *repository.Script) *ScriptManager {
	return &ScriptManager{Script: r}
}
