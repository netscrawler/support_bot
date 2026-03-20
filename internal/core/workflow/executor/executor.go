package executor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"support_bot/internal/core/workflow/execution"
	"support_bot/internal/core/workflow/registry"
	"support_bot/internal/core/workflow/scheduler"
	"sync"
)

// Executor reads ready nodes from the Scheduler and runs their actions
// concurrently, bounded by a worker-pool semaphore.
//
// Error semantics:
//   - If a node's action returns an error, the internal context is cancelled.
//   - Nodes that are ready but not yet started after cancellation are marked
//     as skipped and are not executed.
//   - On caller cancellation, all still-waiting nodes are marked as skipped.
//   - Run returns the first node error encountered.
type Executor struct {
	exec    *execution.Execution
	reg     *registry.Registry
	sched   *scheduler.Scheduler
	workers int
	log     *slog.Logger
}

var errReadyQueueClosed = fmt.Errorf("workflow executor: ready queue closed unexpectedly")

// New creates an Executor.
// workers is the maximum number of nodes that may run in parallel (>= 1).
func New(
	exec *execution.Execution,
	reg *registry.Registry,
	sched *scheduler.Scheduler,
	workers int,
	log *slog.Logger,
) *Executor {
	if workers <= 0 {
		workers = 4
	}

	if reg == nil {
		reg = registry.New()
	}

	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &Executor{
		exec:    exec,
		reg:     reg,
		sched:   sched,
		workers: workers,
		log:     log,
	}
}

// Run starts the scheduler, dispatches nodes, and blocks until the workflow
// completes or ctx is cancelled.
//
// It returns nil on full success, or the first error encountered otherwise.
func (e *Executor) Run(ctx context.Context) error {
	// internal cancel lets us abort on the first node error
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// doneCh is closed once every node reaches a terminal state.
	doneCh := make(chan struct{})
	var doneOnce sync.Once
	signalDoneIfComplete := func() {
		if e.exec.IsComplete() {
			doneOnce.Do(func() {
				close(doneCh)
			})
		}
	}

	var (
		firstErr error
		errOnce  sync.Once
		wg       sync.WaitGroup
	)

	sem := make(chan struct{}, e.workers)

	// seed the scheduler with start nodes
	e.sched.Start()
	signalDoneIfComplete()

	readyCh := e.sched.ReadyCh()

	for {
		select {
		// outer context cancelled (caller-initiated)
		case <-ctx.Done():
			cancel()
			e.exec.SkipAllWaiting(ctx.Err())
			signalDoneIfComplete()
			wg.Wait()

			if firstErr != nil {
				return firstErr
			}

			return fmt.Errorf("workflow executor: %w", ctx.Err())

		// internal abort (node error) or normal completion
		case <-doneCh:
			wg.Wait()

			return firstErr

		// a node is ready to run
		case nodeID, ok := <-readyCh:
			if !ok {
				cancel()
				e.exec.SkipAllWaiting(errReadyQueueClosed)
				signalDoneIfComplete()
				wg.Wait()

				if firstErr != nil {
					return firstErr
				}

				return errReadyQueueClosed
			}

			if runCtx.Err() != nil {
				if e.exec.SkipIfWaiting(nodeID, runCtx.Err()) {
					e.sched.Complete(nodeID)
				}
				signalDoneIfComplete()

				continue
			}

			state := e.exec.GetState(nodeID)
			if state == nil || state.Status != execution.StatusWaiting {
				signalDoneIfComplete()

				continue
			}

			select {
			case sem <- struct{}{}: // acquire worker slot
			case <-runCtx.Done():
				if e.exec.SkipIfWaiting(nodeID, runCtx.Err()) {
					e.sched.Complete(nodeID)
				}
				signalDoneIfComplete()

				continue
			}

			wg.Add(1)

			go func(id string) {
				defer func() {
					<-sem
					wg.Done()
				}()

				if err := e.runNode(runCtx, id); err != nil {
					errOnce.Do(func() {
						firstErr = err
						cancel()
					})
				}

				// always notify the scheduler so children are unblocked
				e.sched.Complete(id)
				signalDoneIfComplete()
			}(nodeID)
		}
	}
}

// runNode executes the action for a single node and updates execution state.
// Returns the action error if one occurred, nil otherwise.
func (e *Executor) runNode(ctx context.Context, nodeID string) (runErr error) {
	node := e.exec.Graph.Nodes[nodeID]
	if node == nil {
		err := fmt.Errorf("workflow executor: node %q not found in runtime graph", nodeID)
		e.log.ErrorContext(ctx, "node not found",
			slog.String("node_id", nodeID),
			slog.Any("error", err),
		)
		e.exec.SetState(nodeID, execution.StatusFailed, nil, err)

		return err
	}

	defer func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("node %q panicked: %v", nodeID, rec)
			e.log.ErrorContext(ctx, "node panicked",
				slog.String("node_id", nodeID),
				slog.Any("panic", rec),
				slog.String("stack", string(debug.Stack())),
			)
			e.exec.SetState(nodeID, execution.StatusFailed, nil, err)
			runErr = err
		}
	}()

	e.log.DebugContext(ctx, "node starting",
		slog.String("node_id", nodeID),
		slog.String("type", node.Type),
	)

	e.exec.SetState(nodeID, execution.StatusRunning, nil, nil)

	// look up the action in the registry
	action, ok := e.reg.Get(node.Type)
	if !ok {
		err := fmt.Errorf("no action registered for type %q (node %q)", node.Type, nodeID)
		e.log.ErrorContext(ctx, "action not found",
			slog.String("node_id", nodeID),
			slog.String("type", node.Type),
		)
		e.exec.SetState(nodeID, execution.StatusFailed, nil, err)

		return err
	}

	input := registry.ActionInput{
		NodeID:  nodeID,
		Context: e.exec.Context,
	}

	resolvedConfig, err := e.exec.Context.ResolveConfig(node.Config)
	if err != nil {
		err = fmt.Errorf("node %q: resolve config: %w", nodeID, err)
		e.log.ErrorContext(ctx, "node config resolve failed",
			slog.String("node_id", nodeID),
			slog.Any("error", err),
		)
		e.exec.SetState(nodeID, execution.StatusFailed, nil, err)

		return err
	}

	input.Config = resolvedConfig

	output, err := action.Execute(ctx, input)
	if err != nil {
		e.log.ErrorContext(ctx, "node failed",
			slog.String("node_id", nodeID),
			slog.Any("error", err),
		)
		e.exec.SetState(nodeID, execution.StatusFailed, nil, err)

		return fmt.Errorf("node %q: %w", nodeID, err)
	}

	e.log.DebugContext(ctx, "node completed",
		slog.String("node_id", nodeID),
	)
	e.exec.SetState(nodeID, execution.StatusCompleted, output.Data, nil)

	return nil
}
