package actions

import (
	"context"
	"fmt"
	"support_bot/internal/core/workflow/definition"
	"support_bot/internal/core/workflow/registry"
)

type PluginActionProvider struct {
	pM pManager
}

func NewPluginActionProvider(pM pManager) *PluginActionProvider {
	return &PluginActionProvider{pM: pM}
}

func (p *PluginActionProvider) GetAction(ctx context.Context, key string) (registry.Action, error) {
	act, err := NewPluginAction(ctx, key, p.pM)
	if err != nil {
		return nil, fmt.Errorf("get plugin action: %w", err)
	}

	return act, nil
}

func (p *PluginActionProvider) RegisterFromDef(ctx context.Context, reg *registry.Registry, def *definition.WorkflowDef) error {
	return RegisterPluginsFromDefinition(ctx, reg, p.pM, def)
}
