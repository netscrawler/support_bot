package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/generator"
	"support_bot/internal/models"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeCollector struct {
	data models.Dataset
}

func (f fakeCollector) Collect(_ context.Context, _ ...models.Card) (models.Dataset, error) {
	return f.data, nil
}

type fakeEvaluator struct {
	approve bool
}

func (f fakeEvaluator) Evaluate(_ context.Context, _ models.Dataset, _ string) (bool, error) {
	return f.approve, nil
}

func (f fakeEvaluator) EvalStr(_ context.Context, expr string) (string, error) {
	return expr, nil
}

func strPtr(s string) *string { return &s }

func newTestGenerator(t *testing.T, approve bool) *generator.Generator {
	t.Helper()

	g := generator.New(
		fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		nil,
		fakeEvaluator{approve: approve},
		1,
		slog.New(slog.DiscardHandler),
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	g.Start(ctx)

	return g
}

func TestOrchestrator_Generate_ReturnsFirstExport(t *testing.T) {
	o := &Orchestrator{gen: newTestGenerator(t, true), log: slog.New(slog.DiscardHandler)}

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: strPtr("out")}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := o.Generate(ctx, report)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if data.FileName != "out_q1.csv" {
		t.Fatalf("Generate() = %+v, want FileName out_q1.csv", data)
	}
}

func TestOrchestrator_Generate_NegativeEvaluationIsNotFound(t *testing.T) {
	o := &Orchestrator{gen: newTestGenerator(t, false), log: slog.New(slog.DiscardHandler)}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := o.Generate(ctx, models.Report{Name: "r1", Evaluation: "false"})
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("Generate() error = %v, want models.ErrNotFound", err)
	}
}

type fakeLoader struct {
	report models.Report
}

func (f fakeLoader) Load(_ context.Context) ([]models.Report, error) { return nil, nil }

func (f fakeLoader) LoadByEvent(_ context.Context, _ string, _ bool) (*models.Report, error) {
	r := f.report

	return &r, nil
}

// blockingCollector.Collect blocks until release is closed, tracking how
// many calls are in flight at once (and the peak observed) so a test can
// assert on concurrency bounds without racing on timing.
type blockingCollector struct {
	release  chan struct{}
	inFlight *int32
	peak     *int32
}

func (b blockingCollector) Collect(_ context.Context, _ ...models.Card) (models.Dataset, error) {
	n := atomic.AddInt32(b.inFlight, 1)

	for {
		p := atomic.LoadInt32(b.peak)
		if n <= p || atomic.CompareAndSwapInt32(b.peak, p, n) {
			break
		}
	}

	<-b.release

	atomic.AddInt32(b.inFlight, -1)

	return models.Dataset{"q1": {{"col": "val"}}}, nil
}

// TestOrchestrator_ProcessGenReportEvent_BoundsInFlightGenerations guards
// against the goroutine-per-event leak: processGenReportEvent must never let
// more than maxInFlightGenerations generateAndDeliver calls run at once,
// however many events land concurrently.
func TestOrchestrator_ProcessGenReportEvent_BoundsInFlightGenerations(t *testing.T) {
	var inFlight, peak int32

	release := make(chan struct{})
	clct := blockingCollector{release: release, inFlight: &inFlight, peak: &peak}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	g := generator.New(clct, nil, fakeEvaluator{approve: false}, maxInFlightGenerations*2, slog.New(slog.DiscardHandler))
	g.Start(ctx)

	o := &Orchestrator{
		gen:    g,
		rL:     fakeLoader{report: models.Report{Name: "r1", Evaluation: "false"}},
		genSem: make(chan struct{}, maxInFlightGenerations),
		log:    slog.New(slog.DiscardHandler),
	}

	var wg sync.WaitGroup
	for range maxInFlightGenerations * 4 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			o.processGenReportEvent(ctx, "ev", true, nil)
		}()
	}

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&inFlight) != maxInFlightGenerations {
		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for %d in-flight generations, got %d",
				maxInFlightGenerations, atomic.LoadInt32(&inFlight),
			)
		}

		time.Sleep(5 * time.Millisecond)
	}

	close(release)
	wg.Wait() // only proves every processGenReportEvent call returned, i.e. was admitted past the semaphore

	deadline = time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&inFlight) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for in-flight generations to drain, got %d", atomic.LoadInt32(&inFlight))
		}

		time.Sleep(5 * time.Millisecond)
	}

	if p := atomic.LoadInt32(&peak); p > maxInFlightGenerations {
		t.Fatalf("peak in-flight generations = %d, want <= %d", p, maxInFlightGenerations)
	}
}
