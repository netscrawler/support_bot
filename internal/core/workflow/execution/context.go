package execution

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// ExecutionContext is the shared, thread-safe key-value store for a single
// workflow execution. Nodes write their outputs here; downstream nodes read
// predecessors' outputs via references of the form "$.node_id" or
// "$.node_id.field.subfield".
type ExecutionContext struct {
	mu      sync.RWMutex
	outputs map[string]any // node_id → output value
}

// NewExecutionContext creates an empty ExecutionContext.
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		outputs: make(map[string]any),
	}
}

// Set stores the output for a node. Overwrites any existing value.
func (c *ExecutionContext) Set(nodeID string, output any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.outputs[nodeID] = output
}

// Get retrieves the output stored for nodeID.
// Returns (nil, false) if no output has been stored yet.
func (c *ExecutionContext) Get(nodeID string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	v, ok := c.outputs[nodeID]

	return v, ok
}

// Resolve evaluates a reference string and returns the referenced value.
//
// Supported formats:
//   - "$.node_id"                 — returns the entire output of node_id
//   - "$.node_id.field.subfield" — traverses nested map keys / struct fields
//
// Returns an error for malformed references or missing values.
func (c *ExecutionContext) Resolve(ref string) (any, error) {
	if !strings.HasPrefix(ref, "$.") {
		return nil, fmt.Errorf("workflow/context: invalid reference %q — must start with $.", ref)
	}

	path := ref[2:]
	if path == "" {
		return nil, fmt.Errorf("workflow/context: invalid reference %q — missing node id", ref)
	}

	parts := strings.SplitN(path, ".", 2)
	if parts[0] == "" {
		return nil, fmt.Errorf("workflow/context: invalid reference %q — empty node id", ref)
	}

	nodeID := parts[0]

	val, ok := c.Get(nodeID)
	if !ok {
		return nil, fmt.Errorf("workflow/context: no output stored for node %q", nodeID)
	}

	// plain node reference — return the whole output
	if len(parts) == 1 {
		return val, nil
	}
	if parts[1] == "" {
		return nil, fmt.Errorf("workflow/context: invalid reference %q — empty field path", ref)
	}

	field, ok := resolveFieldPath(val, parts[1])
	if !ok {
		return nil, fmt.Errorf(
			"workflow/context: field path %q not found in output of node %q (supports nested maps with string keys or struct fields)",
			parts[1], nodeID,
		)
	}

	return field, nil
}

// ResolveConfig returns a copy of raw config where string values that are
// full workflow references ("$.node" / "$.node.field") are replaced with
// values from the execution context.
//
// Resolution runs recursively for nested objects and arrays.
func (c *ExecutionContext) ResolveConfig(raw json.RawMessage) (json.RawMessage, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return raw, nil
	}

	var cfg any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("workflow/context: invalid node config json: %w", err)
	}

	resolved, err := c.resolveConfigValue(cfg, "$")
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(resolved)
	if err != nil {
		return nil, fmt.Errorf("workflow/context: marshal resolved config: %w", err)
	}

	return out, nil
}

func (c *ExecutionContext) resolveConfigValue(v any, path string) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			resolved, err := c.resolveConfigValue(child, path+"."+k)
			if err != nil {
				return nil, err
			}

			out[k] = resolved
		}

		return out, nil
	case []any:
		out := make([]any, len(t))
		for i := range t {
			resolved, err := c.resolveConfigValue(t[i], fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}

			out[i] = resolved
		}

		return out, nil
	case string:
		if !strings.HasPrefix(t, "$.") {
			return t, nil
		}

		resolved, err := c.Resolve(t)
		if err != nil {
			return nil, fmt.Errorf("workflow/context: resolve config at %s: %w", path, err)
		}

		return resolved, nil
	default:
		return v, nil
	}
}

func resolveFieldPath(val any, fieldPath string) (any, bool) {
	v := reflect.ValueOf(val)

	for _, segment := range strings.Split(fieldPath, ".") {
		if segment == "" {
			return nil, false
		}

		var ok bool
		v, ok = resolveFieldSegment(v, segment)
		if !ok {
			return nil, false
		}
	}

	if !v.IsValid() {
		return nil, false
	}

	return v.Interface(), true
}

func resolveFieldSegment(v reflect.Value, segment string) (reflect.Value, bool) {
	v, ok := indirectValue(v)
	if !ok {
		return reflect.Value{}, false
	}

	if v.Kind() == reflect.Map {
		if v.Type().Key().Kind() != reflect.String {
			return reflect.Value{}, false
		}

		mv := v.MapIndex(reflect.ValueOf(segment))
		if !mv.IsValid() {
			return reflect.Value{}, false
		}

		return mv, true
	}

	if v.Kind() == reflect.Struct {
		f := v.FieldByName(segment)
		if !f.IsValid() || !f.CanInterface() {
			return reflect.Value{}, false
		}

		return f, true
	}

	return reflect.Value{}, false
}

func indirectValue(v reflect.Value) (reflect.Value, bool) {
	if !v.IsValid() {
		return reflect.Value{}, false
	}

	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}, false
		}

		v = v.Elem()
	}

	return v, true
}
