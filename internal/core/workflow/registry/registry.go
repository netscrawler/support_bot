package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"support_bot/internal/core/workflow/execution"
	"sync"
)

// ActionInput is the bundle of information passed to an Action when a node
// is executed.
type ActionInput struct {
	// NodeID is the ID of the node being executed.
	NodeID string

	// Config is the raw JSON configuration from the node's definition.
	// The engine resolves workflow references (e.g. "$.collect.result") before
	// action execution; action types are responsible for unmarshalling config.
	Config json.RawMessage

	// Context provides read access to outputs produced by predecessor nodes.
	// Use Context.Get(nodeID) or Context.Resolve("$.nodeID.field.subfield").
	// Resolve field access supports nested outputs that are maps with string keys
	// and structs with exported fields.
	Context *execution.ExecutionContext
}

// ActionOutput is the value returned by an Action on success.
// Data is stored in the ExecutionContext under the node's ID and becomes
// available to downstream nodes.
type ActionOutput struct {
	Data any
}

// Action is the interface that every workflow action must implement —
// both built-in actions and plugin-backed actions.
type Action interface {
	Execute(ctx context.Context, input ActionInput) (ActionOutput, error)
}

// Registry maps action type strings to their Action implementations.
// It is safe for concurrent use after the initial registration phase.
type Registry struct {
	mu      sync.RWMutex
	actions map[string]Action
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{
		actions: make(map[string]Action),
	}
}

// Register adds an action for the given type string.
// It panics if the type is already registered — this is intentional so that
// misconfiguration is caught at startup.
func (r *Registry) Register(actionType string, action Action) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := validateRegistrationInput(actionType, action)
	if err != nil {
		return err
	}

	if _, exists := r.actions[actionType]; exists {
		return fmt.Errorf("workflow/registry: action type %q already registered", actionType)
	}

	r.actions[actionType] = action

	return nil
}

// RegisterOrReplace adds or replaces an action without panicking.
// Intended for testing and hot-reload scenarios.
func (r *Registry) RegisterOrReplace(actionType string, action Action) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := validateRegistrationInput(actionType, action)
	if err != nil {
		return err
	}

	r.actions[actionType] = action

	return nil
}

func validateRegistrationInput(actionType string, action Action) error {
	if strings.TrimSpace(actionType) == "" {
		return fmt.Errorf("workflow/registry: action type must not be empty")
	}

	if action == nil {
		return fmt.Errorf("workflow/registry: action for type %q must not be nil", actionType)
	}

	return nil
}

// Get returns the Action registered for actionType.
// Returns (nil, false) if no action is registered for that type.
func (r *Registry) Get(actionType string) (Action, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.actions[actionType]

	return a, ok
}

// Has reports whether an action is registered for actionType.
func (r *Registry) Has(actionType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.actions[actionType]

	return ok
}

func (r *Registry) Clone() *Registry {
	r.mu.RLock()
	mp := maps.Clone(r.actions)
	r.mu.RUnlock()

	return &Registry{
		actions: mp,
	}
}
