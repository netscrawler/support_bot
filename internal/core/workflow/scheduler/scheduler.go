package scheduler

import (
	"fmt"
	"sort"
	"support_bot/internal/core/workflow/execution"
	"sync"
)

// Scheduler tracks per-node dependency counters and emits IDs of nodes
// that are ready to run (all parents completed) into a buffered channel.
//
// Flow:
//  1. Create with New(exec, bufSize).
//  2. Call Start() to seed the channel with start nodes (in-degree == 0).
//  3. After each node finishes, call Complete(nodeID).
//     This decrements pending counters for children. A child is enqueued only
//     when all parents completed successfully; otherwise it is marked skipped.
//  4. Read from ReadyCh() in the executor.
type Scheduler struct {
	exec *execution.Execution

	mu      sync.Mutex
	pending map[string]int // node_id → remaining unfinished parents
	blocked map[string]bool

	readyCh chan string
}

var errUpstreamNotCompleted = fmt.Errorf("workflow/scheduler: skipped because at least one upstream node was not completed")

var errReadyQueueDropped = fmt.Errorf("workflow/scheduler: skipped because executor stopped consuming ready queue")

// New creates a Scheduler for the given Execution.
//
// bufSize should be at least len(exec.Graph.Nodes) to guarantee that
// Complete never blocks when enqueueing children.
func New(exec *execution.Execution, bufSize int) *Scheduler {
	if bufSize < len(exec.Graph.Nodes) {
		bufSize = len(exec.Graph.Nodes)
	}

	pending := make(map[string]int, len(exec.Graph.Nodes))
	blocked := make(map[string]bool, len(exec.Graph.Nodes))

	for id, node := range exec.Graph.Nodes {
		pending[id] = node.InDegree
		blocked[id] = false
	}

	return &Scheduler{
		exec:    exec,
		pending: pending,
		blocked: blocked,
		readyCh: make(chan string, bufSize),
	}
}

// Start enqueues all start nodes (in-degree == 0) into the ready channel.
// Must be called exactly once before the executor begins reading.
func (s *Scheduler) Start() {
	startIDs := append([]string(nil), s.exec.Graph.StartIDs...)
	sort.Strings(startIDs)

	for _, id := range startIDs {
		s.readyCh <- id
	}
}

// Complete notifies the scheduler that nodeID reached a terminal state.
// For every child, pending parents are decremented. Children with at least one
// non-completed parent are marked skipped and never enqueued.
//
// Complete is safe to call from multiple goroutines concurrently.
func (s *Scheduler) Complete(nodeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	type completion struct {
		id        string
		completed bool
	}

	queue := []completion{{id: nodeID, completed: s.isCompletedLocked(nodeID)}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		node := s.exec.Graph.Nodes[cur.id]
		if node == nil {
			continue
		}

		seenChildren := make(map[string]struct{}, len(node.Children))

		for _, childID := range node.Children {
			if _, seen := seenChildren[childID]; seen {
				continue
			}

			seenChildren[childID] = struct{}{}

			if s.pending[childID] <= 0 {
				continue
			}

			s.pending[childID]--
			if !cur.completed {
				s.blocked[childID] = true
			}

			if s.pending[childID] != 0 {
				continue
			}

			if s.blocked[childID] {
				s.exec.SetState(childID, execution.StatusSkipped, nil, errUpstreamNotCompleted)
				queue = append(queue, completion{id: childID, completed: false})
				continue
			}

			if !s.enqueueReadyLocked(childID) {
				s.exec.SetState(childID, execution.StatusSkipped, nil, errReadyQueueDropped)
				queue = append(queue, completion{id: childID, completed: false})
			}
		}
	}
}

// ReadyCh returns the read-only end of the ready-node channel.
func (s *Scheduler) ReadyCh() <-chan string {
	return s.readyCh
}

func (s *Scheduler) isCompletedLocked(nodeID string) bool {
	st := s.exec.GetState(nodeID)

	return st != nil && st.Status == execution.StatusCompleted
}

func (s *Scheduler) enqueueReadyLocked(nodeID string) bool {
	// Non-blocking send: if executor has already stopped consuming we avoid deadlock.
	select {
	case s.readyCh <- nodeID:
		return true
	default:
		return false
	}
}
