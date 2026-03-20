package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"support_bot/internal/core/workflow/definition"
	"support_bot/internal/core/workflow/registry"
	plugins "support_bot/internal/plugin"
)

var ErrPluginNameVerify = errors.New("plugin name verify")

const pluginTypePrefix = "plugin@"

type pManager interface {
	NewPlugin(ctx context.Context, name string, version *string) (*plugins.LuaPlugin, error)
}

type pluginRunner interface {
	Execute(ctx context.Context, params map[string]any) ([]byte, error)
}

type PluginAction struct {
	plugin pluginRunner
}

func NewPluginAction(ctx context.Context, pNameVer string, manager pManager) (*PluginAction, error) {
	name, ver, err := resolvePluginNameVer(pNameVer)
	if err != nil {
		return nil, err
	}

	plug, err := manager.NewPlugin(ctx, name, &ver)
	if err != nil {
		return nil, err
	}

	if plug == nil {
		return nil, errors.New("plugin action: manager returned nil plugin")
	}

	return &PluginAction{plugin: plug}, nil
}

// RegisterPluginsFromRaw parses workflow JSON and registers actions for all
// plugin-backed nodes (type pattern: "plugin@<name>@<version>").
func RegisterPluginsFromRaw(ctx context.Context, reg *registry.Registry, manager pManager, raw json.RawMessage) error {
	def, err := definition.Parse(raw)
	if err != nil {
		return fmt.Errorf("plugin action: parse workflow: %w", err)
	}

	return RegisterPluginsFromDefinition(ctx, reg, manager, def)
}

// RegisterPluginsFromDefinition registers plugin-backed node types in registry.
func RegisterPluginsFromDefinition(ctx context.Context, reg *registry.Registry, manager pManager, def *definition.WorkflowDef) error {
	if reg == nil {
		return errors.New("plugin action: registry is nil")
	}

	if manager == nil {
		return errors.New("plugin action: manager is nil")
	}

	if def == nil {
		return errors.New("plugin action: workflow definition is nil")
	}

	for _, node := range def.Nodes {
		if !strings.HasPrefix(node.Type, pluginTypePrefix) {
			continue
		}

		if reg.Has(node.Type) {
			continue
		}

		nameVer := strings.TrimPrefix(node.Type, pluginTypePrefix)
		if strings.TrimSpace(nameVer) == "" {
			return fmt.Errorf("plugin action: node %q has empty plugin name/version", node.ID)
		}

		action, err := NewPluginAction(ctx, nameVer, manager)
		if err != nil {
			return fmt.Errorf("plugin action: register %q for node %q: %w", node.Type, node.ID, err)
		}

		if err := reg.Register(node.Type, action); err != nil {
			return fmt.Errorf("plugin action: register %q for node %q: %w", node.Type, node.ID, err)
		}
	}

	return nil
}

func (p *PluginAction) Execute(ctx context.Context, input registry.ActionInput) (registry.ActionOutput, error) {
	if p.plugin == nil {
		return registry.ActionOutput{}, errors.New("plugin action: plugin is nil")
	}

	params := map[string]any{}
	if len(input.Config) > 0 {
		if err := json.Unmarshal(input.Config, &params); err != nil {
			return registry.ActionOutput{}, fmt.Errorf("plugin action: decode config: %w", err)
		}
	}

	out, err := p.plugin.Execute(ctx, params)
	if err != nil {
		return registry.ActionOutput{}, err
	}

	var data any
	if err := json.Unmarshal(out, &data); err != nil {
		return registry.ActionOutput{}, fmt.Errorf("plugin action: decode output: %w", err)
	}

	return registry.ActionOutput{Data: data}, nil
}

func resolvePluginNameVer(pNameVer string) (string, string, error) {
	nameVer := strings.Split(pNameVer, "@")
	if len(nameVer) != 2 {
		return "", "", fmt.Errorf("%w, name and version sig must be name@vX.X.X", ErrPluginNameVerify)
	}

	name, ver := nameVer[0], nameVer[1]
	err := versionValidate(ver)
	if err != nil {
		return "", "", fmt.Errorf("%w: %s : %w", ErrPluginNameVerify, ver, err)
	}

	return name, ver, nil
}

func versionValidate(ver string) error {
	const reg = `^(latest|v\d+(?:\.\d+)*)$`

	exp, err := regexp.Compile(reg)
	if err != nil {
		return err
	}

	if !exp.MatchString(ver) {
		return errors.New("plugin version does not match " + reg)
	}

	return nil
}
