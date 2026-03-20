package graph

import (
	"sort"

	"support_bot/internal/core/workflow/definition"
)

// Build converts a validated WorkflowDef into an immutable RuntimeGraph.
// It runs in O(V+E) and assumes the definition has already been validated.
func Build(def *definition.WorkflowDef) *RuntimeGraph {
	nodes := make(map[string]*RuntimeNode, len(def.Nodes))
	seenEdges := make(map[string]struct{}, len(def.Edges))

	// initialise all nodes
	for _, nd := range def.Nodes {
		nodes[nd.ID] = &RuntimeNode{
			ID:       nd.ID,
			Type:     nd.Type,
			Config:   nd.Config,
			Parents:  make([]string, 0),
			Children: make([]string, 0),
		}
	}

	// wire up edges — build adjacency lists and compute in-degrees
	for _, edge := range def.Edges {
		edgeKey := edge.From + "\x00" + edge.To
		if _, exists := seenEdges[edgeKey]; exists {
			continue
		}

		seenEdges[edgeKey] = struct{}{}

		from := nodes[edge.From]
		to := nodes[edge.To]

		from.Children = append(from.Children, edge.To)
		to.Parents = append(to.Parents, edge.From)
		to.InDegree++
	}

	// collect start nodes (in-degree == 0)
	startIDs := make([]string, 0)

	for id, node := range nodes {
		if node.InDegree == 0 {
			startIDs = append(startIDs, id)
		}

		sort.Strings(node.Parents)
		sort.Strings(node.Children)
	}

	sort.Strings(startIDs)

	return &RuntimeGraph{
		Nodes:    nodes,
		StartIDs: startIDs,
	}
}
