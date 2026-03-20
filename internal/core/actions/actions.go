package actions

// ActionType constants for built-in and plugin actions.
// Use these strings when populating a WorkflowDef's NodeDef.Type field.
//
// Built-in format:   "std@<action>"
// Plugin format:     "plugin@<plugin_name>"
const (
	// Built-in action types.
	ActionTypeCollect  = "std@collect"
	ActionTypeEvaluate = "std@evaluate"
	ActionTypeExport   = "std@export"
	ActionTypeSend     = "std@send"

	// DefaultWorkflow is the well-known type for the legacy linear pipeline.
	DefaultWorkflow = "std@default"

	// PluginCustom is the base type prefix for plugin-backed actions.
	PluginCustom = "plugin@custom"
)
