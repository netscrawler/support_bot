package execution

import (
	"sort"
	"support_bot/internal/core/workflow/graph"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Execution is a single in-flight run of a workflow.
// It holds the immutable runtime graph, per-node states, and the shared
// ExecutionContext through which nodes exchange data.
//
// All public methods are safe for concurrent use.
type Execution struct {
	// ID is a unique identifier for this execution run.
	ID string

	// WorkflowID is the ID taken from the WorkflowDef.
	WorkflowID string

	// Graph is the immutable runtime DAG for this run.
	Graph *graph.RuntimeGraph

	// Context is the shared data store used to pass outputs between nodes.
	Context *ExecutionContext

	mu          sync.RWMutex
	states      map[string]*NodeState // node_id → mutable state
	history     []NodeHistoryEntry
	nextEventID int64
}

// New creates a fresh Execution for the given workflow graph.
// All nodes start in StatusWaiting.
func New(workflowID string, g *graph.RuntimeGraph) *Execution {
	states := make(map[string]*NodeState, len(g.Nodes))

	for id := range g.Nodes {
		states[id] = &NodeState{
			NodeID: id,
			Status: StatusWaiting,
		}
	}

	return &Execution{
		ID:          uuid.NewString(),
		WorkflowID:  workflowID,
		Graph:       g,
		Context:     NewExecutionContext(),
		states:      states,
		history:     make([]NodeHistoryEntry, 0, len(g.Nodes)*2),
		nextEventID: 1,
	}
}

// SetState updates the state for the given node and stores completed-node
// output in the shared ExecutionContext so downstream nodes can read it.
func (e *Execution) SetState(nodeID string, status NodeStatus, output any, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	s, ok := e.states[nodeID]
	if !ok {
		return
	}

	s.Status = status
	s.Output = output
	s.Err = err
	e.appendHistoryLocked(nodeID, status, err)

	if status == StatusCompleted {
		e.Context.Set(nodeID, output)
	}
}

// SkipIfWaiting moves a node from StatusWaiting to StatusSkipped.
// Returns true when the transition was applied.
func (e *Execution) SkipIfWaiting(nodeID string, err error) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	s, ok := e.states[nodeID]
	if !ok || s.Status != StatusWaiting {
		return false
	}

	s.Status = StatusSkipped
	s.Output = nil
	s.Err = err
	e.appendHistoryLocked(nodeID, StatusSkipped, err)

	return true
}

// SkipAllWaiting marks all not-started nodes as skipped.
// Returns the number of transitioned nodes.
func (e *Execution) SkipAllWaiting(err error) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	count := 0
	for _, s := range e.states {
		if s.Status != StatusWaiting {
			continue
		}

		s.Status = StatusSkipped
		s.Output = nil
		s.Err = err
		e.appendHistoryLocked(s.NodeID, StatusSkipped, err)
		count++
	}

	return count
}

// GetState returns a copy-by-value snapshot of the node's state.
// Returns nil if the node ID is unknown.
func (e *Execution) GetState(nodeID string) *NodeState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	s, ok := e.states[nodeID]
	if !ok {
		return nil
	}

	// return a shallow copy to prevent callers from mutating internal state
	cp := *s

	return &cp
}

// IsComplete returns true when every node has reached a terminal status
// (Completed, Failed, or Skipped).
func (e *Execution) IsComplete() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, s := range e.states {
		if !s.Status.IsTerminal() {
			return false
		}
	}

	return true
}

// HasFailed returns true if at least one node ended in StatusFailed.
func (e *Execution) HasFailed() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, s := range e.states {
		if s.Status == StatusFailed {
			return true
		}
	}

	return false
}

// FailedNodes returns the IDs of all nodes that ended in StatusFailed.
func (e *Execution) FailedNodes() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var failed []string

	for id, s := range e.states {
		if s.Status == StatusFailed {
			failed = append(failed, id)
		}
	}

	return failed
}

// History returns a copy of node transition records in append order.
func (e *Execution) History() []NodeHistoryEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]NodeHistoryEntry, len(e.history))
	copy(out, e.history)

	return out
}

// HistoryGraph returns execution history as a graph where events are nodes and
// edges describe per-node sequence and parent->child dependencies.
func (e *Execution) HistoryGraph() HistoryGraph {
	e.mu.RLock()
	events := make([]NodeHistoryEntry, len(e.history))
	copy(events, e.history)
	g := e.Graph
	e.mu.RUnlock()

	edges := make([]HistoryEdge, 0)
	byNode := make(map[string][]NodeHistoryEntry, len(events))

	for _, ev := range events {
		byNode[ev.NodeID] = append(byNode[ev.NodeID], ev)
	}

	nodeIDs := make([]string, 0, len(byNode))
	for id := range byNode {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	firstEvent := make(map[string]int64, len(byNode))
	terminalEvent := make(map[string]int64, len(byNode))

	for _, nodeID := range nodeIDs {
		evs := byNode[nodeID]
		sort.Slice(evs, func(i, j int) bool {
			return evs[i].EventID < evs[j].EventID
		})
		byNode[nodeID] = evs

		if len(evs) > 0 {
			firstEvent[nodeID] = evs[0].EventID
		}

		for i := 1; i < len(evs); i++ {
			edges = append(edges, HistoryEdge{
				FromEventID: evs[i-1].EventID,
				ToEventID:   evs[i].EventID,
				Type:        HistoryEdgeNodeSequence,
			})
		}

		for i := len(evs) - 1; i >= 0; i-- {
			if evs[i].Status.IsTerminal() {
				terminalEvent[nodeID] = evs[i].EventID
				break
			}
		}

		if terminalEvent[nodeID] == 0 {
			terminalEvent[nodeID] = evs[len(evs)-1].EventID
		}
	}

	if g != nil {
		graphNodeIDs := make([]string, 0, len(g.Nodes))
		for nodeID := range g.Nodes {
			graphNodeIDs = append(graphNodeIDs, nodeID)
		}
		sort.Strings(graphNodeIDs)

		for _, nodeID := range graphNodeIDs {
			node := g.Nodes[nodeID]
			for _, childID := range node.Children {
				fromID, okFrom := terminalEvent[nodeID]
				toID, okTo := firstEvent[childID]
				if !okFrom || !okTo {
					continue
				}

				edges = append(edges, HistoryEdge{
					FromEventID: fromID,
					ToEventID:   toID,
					Type:        HistoryEdgeDependency,
				})
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromEventID == edges[j].FromEventID {
			if edges[i].ToEventID == edges[j].ToEventID {
				return edges[i].Type < edges[j].Type
			}

			return edges[i].ToEventID < edges[j].ToEventID
		}

		return edges[i].FromEventID < edges[j].FromEventID
	})

	return HistoryGraph{Events: events, Edges: edges}
}

func (e *Execution) appendHistoryLocked(nodeID string, status NodeStatus, err error) {
	rec := NodeHistoryEntry{
		EventID:   e.nextEventID,
		NodeID:    nodeID,
		Status:    status,
		UpdatedAt: time.Now().UTC(),
	}

	if err != nil {
		rec.Error = err.Error()
	}

	e.history = append(e.history, rec)
	e.nextEventID++
}
