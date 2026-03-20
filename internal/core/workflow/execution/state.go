package execution

import "time"

// NodeStatus is the execution status of a single workflow node.
type NodeStatus string

const (
	// StatusWaiting means the node has not yet started
	// (waiting for parent nodes to complete).
	StatusWaiting NodeStatus = "waiting"

	// StatusRunning means the node's action is currently executing.
	StatusRunning NodeStatus = "running"

	// StatusCompleted means the node finished successfully.
	StatusCompleted NodeStatus = "completed"

	// StatusFailed means the node's action returned an error.
	StatusFailed NodeStatus = "failed"

	// StatusSkipped means the node was not executed
	// (e.g. because the workflow was aborted before it could run).
	StatusSkipped NodeStatus = "skipped"
)

// IsTerminal reports whether status means the node can no longer transition.
func (s NodeStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusSkipped
}

// NodeState holds the runtime state of a single node within an Execution.
type NodeState struct {
	NodeID string
	Status NodeStatus

	// Output is the value returned by the action (nil if not yet completed).
	Output any

	// Err is set when Status == StatusFailed.
	Err error
}

// NodeHistoryEntry stores one status transition record for a node.
type NodeHistoryEntry struct {
	EventID   int64
	NodeID    string
	Status    NodeStatus
	Error     string
	UpdatedAt time.Time
}

type HistoryEdgeType string

const (
	HistoryEdgeNodeSequence HistoryEdgeType = "node_sequence"
	HistoryEdgeDependency   HistoryEdgeType = "dependency"
)

// HistoryEdge links two history events.
type HistoryEdge struct {
	FromEventID int64
	ToEventID   int64
	Type        HistoryEdgeType
}

// HistoryGraph is a graph projection of execution history.
type HistoryGraph struct {
	Events []NodeHistoryEntry
	Edges  []HistoryEdge
}
