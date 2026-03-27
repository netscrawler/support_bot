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
	ActionTypeStart    = "std@start"
	ActionTypeEnd      = "std@end"

	// DefaultWorkflow is the well-known type for the legacy linear pipeline.
	DefaultWorkflow = "std@default"

	// PluginCustom is the base type prefix for plugin-backed actions.
	PluginCustom = "plugin@custom"

	// ActionTypeQuery loads collected data into an in-memory DuckDB instance and
	// executes one or more SQL queries against it. Each named query result is
	// stored as a key in the output map and is available to downstream nodes via
	// the execution context (e.g. "$.query_node.totals").
	ActionTypeQuery = "std@query"

	ActionTypeSave = "std@save"
)
