package definition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

const (
	PseudoStartNodeID = "start"
	PseudoEndNodeID   = "end"

	PseudoStartActionType = "std@pseudo_start"
	PseudoEndActionType   = "std@pseudo_end"
)

// WorkflowDef is the JSON representation of a workflow.
// Stored in Report.Workflow as json.RawMessage.
type WorkflowDef struct {
	ID       string            `json:"id"`
	Version  string            `json:"version,omitempty"`
	Nodes    []NodeDef         `json:"nodes"`
	Edges    []EdgeDef         `json:"edges"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (wd *WorkflowDef) AppendMetadata(meta ...map[string]string) {
	if len(meta) == 0 {
		return
	}

	if wd.Metadata == nil {
		wd.Metadata = make(map[string]string, len(meta))
	}

	for _, m := range meta {
		maps.Insert(wd.Metadata, maps.All(m))
	}
}

// NodeDef describes a single node in the workflow graph.
type NodeDef struct {
	// ID must be unique within the workflow.
	ID string `json:"id"`
	// Type identifies the action to execute, e.g. "std@collect", "plugin@my_plugin@v1.2.3".
	Type string `json:"type"`
	// Config is arbitrary JSON passed to the action at runtime.
	Config json.RawMessage `json:"config,omitempty"`
}

func (n *NodeDef) IsPlugin() bool {
	return strings.HasPrefix(n.Type, "plugin@")
}

func (n *NodeDef) GetTypeName() string {
	if !n.IsPlugin() {
		return ""
	}

	return strings.Split(n.Type, "@")[1]
}

// EdgeDef is a directed connection from one node to another.
// Semantics: node To runs only after node From completes.
type EdgeDef struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Parse unmarshals raw JSON bytes into a WorkflowDef.
func Parse(raw []byte) (*WorkflowDef, error) {
	var def WorkflowDef
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&def); err != nil {
		return nil, fmt.Errorf("workflow/definition: parse: %w", err)
	}

	if err := def.EnsurePseudoBoundaryNodes(); err != nil {
		return nil, fmt.Errorf("workflow/definition: normalize: %w", err)
	}

	return &def, nil
}

func (wd *WorkflowDef) EnsurePseudoBoundaryNodes() error {
	if wd == nil {
		return fmt.Errorf("workflow definition is nil")
	}

	nodeByID := make(map[string]NodeDef, len(wd.Nodes))
	for _, n := range wd.Nodes {
		nodeByID[n.ID] = n
	}

	if n, ok := nodeByID[PseudoStartNodeID]; ok && n.Type != PseudoStartActionType {
		return fmt.Errorf("node %q is reserved for %q", PseudoStartNodeID, PseudoStartActionType)
	}

	if n, ok := nodeByID[PseudoEndNodeID]; ok && n.Type != PseudoEndActionType {
		return fmt.Errorf("node %q is reserved for %q", PseudoEndNodeID, PseudoEndActionType)
	}

	filteredEdges := make([]EdgeDef, 0, len(wd.Edges)+len(wd.Nodes)*2)
	autoEdgeSet := make(map[string]struct{}, len(wd.Nodes)*2)
	existingEdgeSet := make(map[string]struct{}, len(wd.Edges))
	inDegree := make(map[string]int, len(wd.Nodes))
	outDegree := make(map[string]int, len(wd.Nodes))

	addAutoEdge := func(from, to string) {
		k := from + "\x00" + to
		if _, exists := existingEdgeSet[k]; exists {
			return
		}

		if _, exists := autoEdgeSet[k]; exists {
			return
		}

		autoEdgeSet[k] = struct{}{}
		filteredEdges = append(filteredEdges, EdgeDef{From: from, To: to})
	}

	for _, e := range wd.Edges {
		if e.From == PseudoStartNodeID || e.To == PseudoStartNodeID || e.From == PseudoEndNodeID || e.To == PseudoEndNodeID {
			continue
		}

		filteredEdges = append(filteredEdges, e)
		existingEdgeSet[e.From+"\x00"+e.To] = struct{}{}
		inDegree[e.To]++
		outDegree[e.From]++
	}

	realNodeIDs := make([]string, 0, len(wd.Nodes))
	hasStart := false
	hasEnd := false

	for _, n := range wd.Nodes {
		switch n.ID {
		case PseudoStartNodeID:
			hasStart = true
		case PseudoEndNodeID:
			hasEnd = true
		default:
			realNodeIDs = append(realNodeIDs, n.ID)
		}
	}

	if !hasStart {
		wd.Nodes = append(wd.Nodes, NodeDef{ID: PseudoStartNodeID, Type: PseudoStartActionType})
	}

	if !hasEnd {
		wd.Nodes = append(wd.Nodes, NodeDef{ID: PseudoEndNodeID, Type: PseudoEndActionType})
	}

	terminalNodeIDs := make([]string, 0, len(realNodeIDs))

	if len(realNodeIDs) == 0 {
		addAutoEdge(PseudoStartNodeID, PseudoEndNodeID)
	} else {
		for _, id := range realNodeIDs {
			if inDegree[id] == 0 {
				addAutoEdge(PseudoStartNodeID, id)
			}

			if outDegree[id] == 0 {
				terminalNodeIDs = append(terminalNodeIDs, id)
				addAutoEdge(id, PseudoEndNodeID)
			}
		}
	}

	wd.Edges = filteredEdges

	for i := range wd.Nodes {
		if wd.Nodes[i].ID != PseudoEndNodeID {
			continue
		}

		cfg, err := json.Marshal(struct {
			TerminalNodeIDs []string `json:"terminal_node_ids,omitempty"`
		}{TerminalNodeIDs: terminalNodeIDs})
		if err != nil {
			return fmt.Errorf("marshal end config: %w", err)
		}

		wd.Nodes[i].Config = cfg
		break
	}

	return nil
}
