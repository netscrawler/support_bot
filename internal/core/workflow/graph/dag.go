package graph

import "encoding/json"

// RuntimeGraph is the in-memory DAG built from a WorkflowDef.
// It is immutable after construction and safe for concurrent reads.
type RuntimeGraph struct {
	// Nodes maps node ID → node descriptor.
	Nodes map[string]*RuntimeNode
	// StartIDs holds the IDs of all nodes with in-degree == 0.
	// These are the entry points of the workflow.
	StartIDs []string
}

// RuntimeNode is a single vertex in the RuntimeGraph.
type RuntimeNode struct {
	ID   string
	Type string // action type, e.g. "std@collect", "plugin@my_plugin"

	// Config is passed verbatim to the action at execution time.
	Config json.RawMessage

	// Parents and Children are node IDs (adjacency list).
	Parents  []string
	Children []string

	// InDegree is the number of parents (used by the scheduler).
	InDegree int
}
