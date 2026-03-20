package validator

import (
	"errors"
	"fmt"
	"support_bot/internal/core/workflow/definition"
)

// Sentinel errors returned by Validate.
var (
	ErrNilWorkflowDef     = errors.New("workflow definition is nil")
	ErrEmptyNodes         = errors.New("workflow has no nodes")
	ErrNoStartNodes       = errors.New("workflow has no start nodes (all nodes have parents — possible cycle)")
	ErrDuplicateNodeID    = errors.New("duplicate node id")
	ErrDuplicateEdge      = errors.New("duplicate edge")
	ErrUnknownEdgeRef     = errors.New("edge references unknown node id")
	ErrCycleDetected      = errors.New("cycle detected in workflow graph")
	ErrEmptyNodeType      = errors.New("node has empty type")
	ErrEmptyNodeID        = errors.New("node has empty id")
	ErrMissingPseudoStart = errors.New("workflow has no pseudo start node")
	ErrMissingPseudoEnd   = errors.New("workflow has no pseudo end node")
	ErrInvalidPseudoType  = errors.New("pseudo boundary node has invalid type")
)

// Validate checks the structural integrity of a WorkflowDef:
//  1. all node IDs are non-empty and unique
//  2. all node types are non-empty
//  3. edge references point to existing node IDs
//  4. no cycles (Kahn's algorithm, O(V+E))
func Validate(def *definition.WorkflowDef) error {
	if def == nil {
		return ErrNilWorkflowDef
	}

	if len(def.Nodes) == 0 {
		return ErrEmptyNodes
	}

	// --- 1 & 2: unique IDs, non-empty fields ---
	ids := make(map[string]struct{}, len(def.Nodes))
	hasPseudoStart := false
	hasPseudoEnd := false

	for _, n := range def.Nodes {
		if n.ID == "" {
			return ErrEmptyNodeID
		}

		if n.Type == "" {
			return fmt.Errorf("%w: node %q", ErrEmptyNodeType, n.ID)
		}

		if _, exists := ids[n.ID]; exists {
			return fmt.Errorf("%w: %q", ErrDuplicateNodeID, n.ID)
		}

		if n.ID == definition.PseudoStartNodeID {
			hasPseudoStart = true
			if n.Type != definition.PseudoStartActionType {
				return fmt.Errorf("%w: node %q", ErrInvalidPseudoType, n.ID)
			}
		}

		if n.ID == definition.PseudoEndNodeID {
			hasPseudoEnd = true
			if n.Type != definition.PseudoEndActionType {
				return fmt.Errorf("%w: node %q", ErrInvalidPseudoType, n.ID)
			}
		}

		ids[n.ID] = struct{}{}
	}

	if !hasPseudoStart {
		return ErrMissingPseudoStart
	}

	if !hasPseudoEnd {
		return ErrMissingPseudoEnd
	}

	// --- 3: edge refs + build adjacency list for cycle check ---
	inDegree := make(map[string]int, len(def.Nodes))
	adj := make(map[string][]string, len(def.Nodes))
	seenEdges := make(map[string]struct{}, len(def.Edges))

	for id := range ids {
		inDegree[id] = 0
	}

	for _, e := range def.Edges {
		if _, ok := ids[e.From]; !ok {
			return fmt.Errorf("%w: from=%q", ErrUnknownEdgeRef, e.From)
		}

		if _, ok := ids[e.To]; !ok {
			return fmt.Errorf("%w: to=%q", ErrUnknownEdgeRef, e.To)
		}

		edgeKey := e.From + "\x00" + e.To
		if _, exists := seenEdges[edgeKey]; exists {
			return fmt.Errorf("%w: %q -> %q", ErrDuplicateEdge, e.From, e.To)
		}

		seenEdges[edgeKey] = struct{}{}

		inDegree[e.To]++
		adj[e.From] = append(adj[e.From], e.To)
	}

	done := isCyclic(def, inDegree, adj)
	if done {
		return ErrCycleDetected
	}

	return nil
}

func isCyclic(def *definition.WorkflowDef, inDegree map[string]int, adj map[string][]string) bool {
	queue := make([]string, 0, len(def.Nodes))

	for id, d := range inDegree {
		if d == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		visited++

		for _, child := range adj[cur] {
			inDegree[child]--

			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	return visited != len(def.Nodes)
}
