# On-Demand Report Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the public "generate report by ID" HTTP endpoint (`GET /api/v1/public/report/{report_id}`) run report generation through the exact same code path and the exact same bounded worker pool as scheduled report generation, instead of a separate/unbounded path — and make the endpoint actually reachable (it currently is not wired into the running app at all).

**Architecture:** `internal/generator.Generator` already owns a fixed-size worker pool (`numWorkers`, currently 4) that consumes `models.Report` jobs off a channel fed by the cron/event orchestrator and runs collect → pipeline → evaluate → export → deliver-to-recipients. We extract the collect/pipeline/evaluate/export part into a shared `generate` method, introduce a `Job` envelope (report + optional result channel) as the channel's element type, and add a `GenerateOnDemand` method that enqueues onto that *same* channel and blocks for the result. Because it's the same channel and the same `numWorkers` goroutines, on-demand requests are rate-limited exactly like scheduled ones, and any burst of either kind queues behind the other. The result is adapted to `service.ReportGenerator` and wired through `service.Report` (already built) into the HTTP handler (already built), then the still-missing HTTP server bring-up (config, routes, lifecycle) is completed so the endpoint is actually served.

**Tech Stack:** Go 1.27, stdlib `net/http`/`net/http.ServeMux`, `database/sql`/`sqlx`, no new dependencies.

**Spec:** No separate spec document — the Architecture section above, plus the per-task Interfaces blocks, are the spec. Two design decisions were confirmed with the user before this plan was written and are locked in as constraints:
- On-demand generation shares the **same** worker pool/channel as scheduled generation (not a separate bounded pool).
- On-demand generation does **not** deliver to the report's configured Recipients — it only returns the exported file(s) to the HTTP caller.

## Global Constraints

- Do not change the observable behavior of scheduled (cron/event) report generation — `deliver` must be byte-for-byte the same logic that exists today in `createReport`.
- On-demand generation must go through `Generator.c` (the shared channel) and `Generator`'s existing worker pool — no second pool, no unbounded goroutines.
- A report with a negative evaluation result (nothing to report) must surface to the on-demand caller as `models.ErrNotFound`, which the HTTP handler already maps to 404 (see `internal/api/http/handlers/get_public_generated_report_by_id.go`).
- Multi-export reports: on-demand generation returns the **first** exported file. This is a known simplification — flag it, don't build a multi-file response format that nobody asked for.
- No new third-party dependencies.
- Every new exported behavior gets a real Go test in the same package (existing convention: plain stdlib `testing`, table-driven where it fits — see `internal/service/report_validator_test.go`). Pure rewiring (channel type changes, config plumbing) is verified by `go build ./...` instead of a new test, since there's no new branching logic to cover.

---

## File Structure

- `internal/generator/generator.go` — **modify**. Extract `generate`/`deliver` out of `createReport`; change the channel element type from `models.Report` to `Job`; branch `worker` on whether a job carries a result channel.
- `internal/generator/on_demand.go` — **new**. `Job`, `JobResult`, `generateOnDemand` (pipeline + not-found mapping), `GenerateOnDemand` (enqueue + wait), `ReportGeneratorAdapter` (adapts `*Generator` to `service.ReportGenerator`).
- `internal/generator/generator_test.go` — **new**. Tests for `generate`, `generateOnDemand`, and the shared-pool behavior of `GenerateOnDemand`.
- `internal/orchestrator/orchestrator.go` — **modify**. `ReportC`/`reportC` become `chan generator.Job`; the two send sites wrap the report in `generator.Job{Report: report}`.
- `internal/service/report_generator.go` — **modify**. Map `models.ErrNotFound` coming back from `gen.Generate` to `errorz.ErrNotFound`, same as the existing DB-lookup branch.
- `internal/service/report_generator_test.go` — **new**. Tests `Report.GenerateReport`'s error mapping with fake `ReportDB`/`ReportGenerator`.
- `internal/api/http/server.go` — **modify**. Expose the router so routes can be registered after construction.
- `internal/api/http/handlers/handler.go` — **modify**. Add `NewHandler` constructor.
- `internal/config/config.go` — **modify**. Add `HTTP http.Config` field and fold its `Validate()` in.
- `internal/config/default.go` — **modify**. Add HTTP defaults.
- `config/config.example.yaml` — **modify**. Document the new `http:` section.
- `internal/app/app.go` — **modify**. Change `reportChan`'s type; build `service.Report` + `generator.ReportGeneratorAdapter`; build and start/stop the HTTP server with routes registered.

---

### Task 1: Extract shared generation core in `Generator`

**Files:**
- Modify: `internal/generator/generator.go:111-231` (the body of `createReport`)
- Test: `internal/generator/generator_test.go` (new)

**Interfaces:**
- Consumes: existing `Generator` struct fields (`clct Collector`, `eval Evaluator`, `proc *processor.Processor`, `snd models.SenderProvider`, `sentMsgRepo SentMsgRepository`, `log *slog.Logger`) and the existing `Collector`/`Evaluator` interfaces — unchanged.
- Produces: `func (g *Generator) generate(ctx context.Context, report models.Report) (data models.Dataset, res []models.Data, approve bool, err error)` and `func (g *Generator) deliver(ctx context.Context, report models.Report, data models.Dataset, res []models.Data) error`, both used by Task 2's worker changes and by the rewritten `createReport`.

- [ ] **Step 1: Write the failing tests**

Create `internal/generator/generator_test.go`:

```go
package generator

import (
	"context"
	"log/slog"
	"support_bot/internal/models"
	"testing"
)

type fakeCollector struct {
	data models.Dataset
	err  error
}

func (f fakeCollector) Collect(_ context.Context, _ ...models.Card) (models.Dataset, error) {
	return f.data, f.err
}

type fakeEvaluator struct {
	approve bool
	err     error
}

func (f fakeEvaluator) Evaluate(_ context.Context, _ models.Dataset, _ string) (bool, error) {
	return f.approve, f.err
}

func (f fakeEvaluator) EvalStr(_ context.Context, expr string) (string, error) {
	return expr, nil
}

func strPtr(s string) *string { return &s }

func TestGenerate_ExportsOnApprove(t *testing.T) {
	g := &Generator{
		clct: fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval: fakeEvaluator{approve: true},
		log:  slog.New(slog.DiscardHandler),
	}

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: strPtr("out")}},
	}

	data, res, approve, err := g.generate(context.Background(), report)
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}

	if !approve {
		t.Fatalf("generate() approve = false, want true")
	}

	if len(res) != 1 || res[0].FileName != "out_q1.csv" {
		t.Fatalf("generate() res = %+v, want one file named out_q1.csv", res)
	}

	if len(data) != 1 {
		t.Fatalf("generate() data = %+v, want the collected dataset back", data)
	}
}

func TestGenerate_NegativeEvaluationSkipsExport(t *testing.T) {
	g := &Generator{
		clct: fakeCollector{data: models.Dataset{}},
		eval: fakeEvaluator{approve: false},
		log:  slog.New(slog.DiscardHandler),
	}

	report := models.Report{Name: "r1", Evaluation: "false"}

	_, res, approve, err := g.generate(context.Background(), report)
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}

	if approve {
		t.Fatalf("generate() approve = true, want false")
	}

	if res != nil {
		t.Fatalf("generate() res = %+v, want nil", res)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/generator/... -run TestGenerate_ -v`
Expected: FAIL to compile — `g.generate undefined (type *Generator has no field or method generate)`

- [ ] **Step 3: Extract `generate` and `deliver` out of `createReport`**

In `internal/generator/generator.go`, replace the existing `createReport` function (lines 111-231) with:

```go
func (g *Generator) createReport(ctx context.Context, report models.Report) error {
	data, res, approve, err := g.generate(ctx, report)
	if err != nil {
		return err
	}

	if !approve {
		return nil
	}

	return g.deliver(ctx, report, data, res)
}

// generate runs the shared report pipeline: resolve query params, collect
// data, run the processing pipeline (if any), evaluate the report
// condition, and export the result. It stops short of delivering anything,
// so both the scheduled path (createReport) and the on-demand path
// (generateOnDemand, added in on_demand.go) can reuse it unchanged.
func (g *Generator) generate(
	ctx context.Context,
	report models.Report,
) (data models.Dataset, res []models.Data, approve bool, err error) {
	l := g.log
	l.DebugContext(ctx, "start generating report", slog.Any("report", report))

	var queries []models.Card

	for _, q := range report.Queries {
		err := q.ResolveParams(ctx, g.eval)
		if err != nil {
			l.ErrorContext(
				ctx,
				"resolving query params",
				slog.Any("error", err),
				slog.Any("query", q),
			)
		}

		queries = append(queries, q)
	}

	data, err = g.clct.Collect(ctx, queries...)
	if err != nil && !errors.Is(err, collector.ErrEmtyCard) {
		l.ErrorContext(ctx, "error while collect data", slog.Any("error", err))

		return nil, nil, false, err
	}

	if report.Pipeline != nil {
		l.InfoContext(
			ctx,
			"start pipeline for report",
			slog.Any("pipeline", report.Pipeline.Name),
			slog.Any("report", report.Name),
		)

		processed, err := g.proc.Process(ctx, data, report.Pipeline)
		if err != nil {
			l.ErrorContext(
				ctx,
				"pipeline execution error, stop generate report",
				slog.Any("error", err),
			)

			return nil, nil, false, err
		}
		data = processed
	}

	approve, err = g.eval.Evaluate(ctx, data, report.Evaluation)
	if err != nil {
		l.ErrorContext(ctx, "error while evaluate report", slog.Any("error", err))

		return nil, nil, false, err
	}

	if !approve {
		l.InfoContext(ctx, "negative result of evaluating, don`t send report")

		return data, nil, false, nil
	}

	res = make([]models.Data, 0, len(report.Exports))

	for _, e := range report.Exports {
		r, err := exporter.Export(data, e)
		if err != nil {
			l.ErrorContext(
				ctx,
				"error while export report",
				slog.Any("error", err),
				slog.Any("export", e),
			)

			continue
		}

		res = append(res, r...)
	}

	return data, res, true, nil
}

// deliver sends a generated report to its configured recipients and records
// the resulting message state. This is the original tail end of
// createReport, unchanged, used only by the scheduled path.
func (g *Generator) deliver(
	ctx context.Context,
	report models.Report,
	data models.Dataset,
	res []models.Data,
) error {
	l := g.log

	if len(report.Recipients) == 0 {
		l.ErrorContext(ctx, "empty targets list")

		return fmt.Errorf("empty targets list")
	}

	// Достаем "_meta" лист из данных, для использования в шаблоне email
	addMeta := make(map[string]any)
	meta, ok := data["_meta"]
	if ok {
		for _, d := range meta {
			maps.Insert(addMeta, maps.All(d))
		}
	}
	l.InfoContext(ctx, "meta", slog.Any("meta", addMeta), slog.Any("_meta", data["_meta"]))
	msg := models.NewMessage(report.Name, res, addMeta, report.Recipients...)

	resMsg, err := msg.Send(ctx, g.snd)
	if err != nil {
		l.ErrorContext(ctx, "error while send message", slog.Any("error", err))
	}

	if len(resMsg) == 0 {
		l.InfoContext(ctx, "report generated")

		return nil
	}

	l.InfoContext(
		ctx,
		"saving message to database",
		slog.Any("report", report.Name),
		slog.Any("message", resMsg),
	)

	err = g.sentMsgRepo.saveTgMsg(ctx, msg.ReportName, resMsg)
	if err != nil {
		l.WarnContext(ctx, "result msg save failed", slog.Any("error", err))
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/generator/... -run TestGenerate_ -v`
Expected: PASS

- [ ] **Step 5: Full package build**

Run: `go build ./internal/generator/...`
Expected: builds clean (the rest of the file — `Collector`, `Evaluator`, `Generator`, `New`, `Start`, `worker` — is untouched by this task, still references `models.Report` on the channel until Task 2).

- [ ] **Step 6: Commit**

```bash
git add internal/generator/generator.go internal/generator/generator_test.go
git commit -m "refactor(generator): extract generate/deliver out of createReport"
```

---

### Task 2: Add the on-demand path, sharing `Generator`'s worker pool

**Files:**
- Create: `internal/generator/on_demand.go`
- Modify: `internal/generator/generator.go:30-31` (struct field), `:48-49` (`New` param), `:82` (`worker` param + body)
- Test: `internal/generator/generator_test.go` (append)

**Interfaces:**
- Consumes: `g.generate` and `g.createReport` from Task 1.
- Produces: `type Job struct { Report models.Report; Result chan<- JobResult }`, `type JobResult struct { Data []models.Data; Err error }`, `func (g *Generator) GenerateOnDemand(ctx context.Context, report models.Report) ([]models.Data, error)`, `type ReportGeneratorAdapter struct { Gen *Generator }` with `func (a ReportGeneratorAdapter) Generate(ctx context.Context, report models.Report) (models.Data, error)`. `Generator.c` is now `chan Job` — Task 3 (orchestrator, app.go) depends on this exact type.

- [ ] **Step 1: Write the failing tests**

Append to `internal/generator/generator_test.go`:

```go
import (
	// ...existing imports...
	"errors"
	"time"
)

func TestGenerateOnDemand_ReturnsExportedFiles(t *testing.T) {
	g := &Generator{
		clct: fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval: fakeEvaluator{approve: true},
		log:  slog.New(slog.DiscardHandler),
	}

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: strPtr("out")}},
	}

	res, err := g.generateOnDemand(context.Background(), report)
	if err != nil {
		t.Fatalf("generateOnDemand() error = %v", err)
	}

	if len(res) != 1 || res[0].FileName != "out_q1.csv" {
		t.Fatalf("generateOnDemand() res = %+v", res)
	}
}

func TestGenerateOnDemand_NegativeEvaluationIsNotFound(t *testing.T) {
	g := &Generator{
		clct: fakeCollector{data: models.Dataset{}},
		eval: fakeEvaluator{approve: false},
		log:  slog.New(slog.DiscardHandler),
	}

	_, err := g.generateOnDemand(context.Background(), models.Report{Name: "r1", Evaluation: "false"})
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("generateOnDemand() error = %v, want models.ErrNotFound", err)
	}
}

func TestGenerateOnDemand_SharesWorkerPoolWithScheduledJobs(t *testing.T) {
	g := &Generator{
		c:          make(chan Job),
		clct:       fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval:       fakeEvaluator{approve: true},
		numWorkers: 1,
		log:        slog.New(slog.DiscardHandler),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g.Start(ctx)

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: strPtr("out")}},
	}

	type outcome struct {
		res []models.Data
		err error
	}

	results := make(chan outcome, 2)

	for range 2 {
		go func() {
			res, err := g.GenerateOnDemand(context.Background(), report)
			results <- outcome{res: res, err: err}
		}()
	}

	for range 2 {
		select {
		case o := <-results:
			if o.err != nil {
				t.Errorf("GenerateOnDemand() error = %v", o.err)
			}

			if len(o.res) != 1 || o.res[0].FileName != "out_q1.csv" {
				t.Errorf("GenerateOnDemand() res = %+v", o.res)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for GenerateOnDemand result")
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/generator/... -run TestGenerateOnDemand_ -v`
Expected: FAIL to compile — `undefined: Job`, `g.generateOnDemand undefined`, `g.GenerateOnDemand undefined`

- [ ] **Step 3: Create `internal/generator/on_demand.go`**

```go
package generator

import (
	"context"
	"support_bot/internal/models"
)

// Job is a unit of work sent through Generator's shared worker pool. Result
// is nil for fire-and-forget scheduled reports (the orchestrator's normal
// flow); when set, the worker writes the outcome there instead of
// delivering to recipients, so on-demand callers share the exact same
// concurrency-limited pool — and the same 5-minute per-job timeout — as
// scheduled generation.
type Job struct {
	Report models.Report
	Result chan<- JobResult
}

type JobResult struct {
	Data []models.Data
	Err  error
}

// generateOnDemand runs the shared report pipeline and returns its exported
// files without delivering them to the report's recipients. A negative
// evaluation result (nothing to report) is surfaced as models.ErrNotFound,
// matching the "not found" semantics the HTTP layer expects.
func (g *Generator) generateOnDemand(ctx context.Context, report models.Report) ([]models.Data, error) {
	_, res, approve, err := g.generate(ctx, report)
	if err != nil {
		return nil, err
	}

	if !approve || len(res) == 0 {
		return nil, models.ErrNotFound
	}

	return res, nil
}

// GenerateOnDemand enqueues report onto the same channel the scheduled
// worker pool consumes and blocks for the result, so an on-demand request
// is bound by the same numWorkers concurrency limit as scheduled
// generation, and queues behind/ahead of scheduled jobs fairly.
func (g *Generator) GenerateOnDemand(ctx context.Context, report models.Report) ([]models.Data, error) {
	result := make(chan JobResult, 1)

	select {
	case g.c <- Job{Report: report, Result: result}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case r := <-result:
		return r.Data, r.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReportGeneratorAdapter adapts Generator to service.ReportGenerator's
// single-file Generate signature (internal/service/report_generator.go).
// On-demand reports are expected to declare exactly one export; if more are
// configured, the first is returned — a known simplification.
type ReportGeneratorAdapter struct {
	Gen *Generator
}

func (a ReportGeneratorAdapter) Generate(ctx context.Context, report models.Report) (models.Data, error) {
	res, err := a.Gen.GenerateOnDemand(ctx, report)
	if err != nil {
		return models.Data{}, err
	}

	if len(res) == 0 {
		return models.Data{}, models.ErrNotFound
	}

	return res[0], nil
}
```

- [ ] **Step 4: Switch the channel element type to `Job` and branch `worker`**

In `internal/generator/generator.go`, change the struct field (was `c chan models.Report`):

```go
type Generator struct {
	c chan Job
	// ...unchanged fields below...
```

Change `New`'s parameter (was `c chan models.Report,`):

```go
func New(
	c chan Job,
	clct Collector,
	// ...unchanged...
```

Replace `worker` (was `func (g *Generator) worker(ctx context.Context, jobs <-chan models.Report, id uint8)`):

```go
func (g *Generator) worker(ctx context.Context, jobs <-chan Job, id uint8) {
	g.log.DebugContext(ctx, fmt.Sprintf("start worker %d", id))

	for {
		select {
		case <-ctx.Done():
			g.log.DebugContext(ctx, "context cancelled")

			return
		case j, ok := <-jobs:
			if !ok {
				g.log.DebugContext(ctx, "jobs chan closed")

				return
			}

			rCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			rvCtx := logger.AppendCtx(rCtx, slog.Any("report_name", j.Report.Name))

			if j.Result != nil {
				res, err := g.generateOnDemand(rvCtx, j.Report)
				j.Result <- JobResult{Data: res, Err: err}
				cancel()

				continue
			}

			err := g.createReport(rvCtx, j.Report)
			if err != nil {
				g.log.ErrorContext(rvCtx, "error create report", slog.Any("error", err))
			}

			cancel()
		}
	}
}
```

`Start` is unchanged (it already just does `go g.worker(ctx, g.c, i)`; the types now line up).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/generator/... -v`
Expected: PASS (all of Task 1's and Task 2's tests)

- [ ] **Step 6: Package build**

Run: `go build ./internal/generator/...`
Expected: builds clean. `go build ./...` will still fail at this point — `internal/orchestrator` and `internal/app` still construct/use `chan models.Report`; that's fixed in Task 3.

- [ ] **Step 7: Commit**

```bash
git add internal/generator/generator.go internal/generator/on_demand.go internal/generator/generator_test.go
git commit -m "feat(generator): add on-demand generation sharing the scheduled worker pool"
```

---

### Task 3: Rewire callers onto `generator.Job` and wire the service layer

**Files:**
- Modify: `internal/orchestrator/orchestrator.go:20` (field), `:34` (param), `:121` (send), `:152` (send)
- Modify: `internal/service/report_generator.go` (error mapping in `GenerateReport`)
- Test: `internal/service/report_generator_test.go` (new)
- Modify: `internal/app/app.go:278` (channel type), `:319-322` (constructing `Generator`/`Orchestrator`, add `service.Report`)

**Interfaces:**
- Consumes: `generator.Job` from Task 2; `repository.Repository.GetPublicReportByID` (already implemented, satisfies `service.ReportDB`); `generator.ReportGeneratorAdapter` (already implemented, satisfies `service.ReportGenerator`); `service.NewReport(db ReportDB, gen ReportGenerator, log *slog.Logger) *Report` (already implemented).
- Produces: a fully wired `reportGenSvc *service.Report` value in `app.go`, consumed by Task 4's HTTP handler wiring.

- [ ] **Step 1: Update `internal/orchestrator/orchestrator.go`**

Change the struct field (was `ReportC chan models.Report`):

```go
type Orchestrator struct {
	EventC        chan models.Event
	SpecialEventC chan models.SpecialEventForLK

	ReportC chan generator.Job
	DeleteC chan models.Event
	// ...unchanged...
```

Add the import:

```go
import (
	"context"
	"log/slog"
	"support_bot/internal/generator"
	"support_bot/internal/models"
	"sync"
	"time"
)
```

Change `New`'s parameter (was `reportC chan models.Report,`):

```go
func New(
	evC chan models.Event,
	specialEventC chan models.SpecialEventForLK,
	reportC chan generator.Job,
	delC chan models.Event,
	rl ReportLoader,
	log *slog.Logger,
) *Orchestrator {
```

Change both send sites — in `processGenReportEvent` (was `case o.ReportC <- report:`):

```go
		select {
		case <-ctx.Done():
			o.log.InfoContext(ctx, "context cancelled. stopping")

			return
		case o.ReportC <- generator.Job{Report: report}:
			o.log.DebugContext(
				ctx,
				"sending report to generator",
				slog.Any("report", report.Name),
			)
		}
```

and identically in `processGenReportSpecialEvent` (same replacement, `o.ReportC <- generator.Job{Report: report}`).

- [ ] **Step 2: Fix `internal/service/report_generator.go`'s error mapping**

The `gen.Generate` error branch currently wraps every error (including a not-found one) as `errorz.ErrInternal`, so a negative-evaluation on-demand report would incorrectly surface as a 500 instead of 404. Change:

```go
	data, err := r.gen.Generate(ctx, *report)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.Data{}, errorz.ErrNotFound
		}

		return models.Data{},
			fmt.Errorf("%w: generate report: %w", errorz.ErrInternal, err)
	}
```

- [ ] **Step 3: Write the test for that mapping**

Create `internal/service/report_generator_test.go`:

```go
package service

import (
	"context"
	"log/slog"
	"support_bot/internal/errorz"
	"support_bot/internal/models"
	"testing"
)

type fakeReportDB struct {
	report *models.Report
	err    error
}

func (f fakeReportDB) GetPublicReportByID(_ context.Context, _ string) (*models.Report, error) {
	return f.report, f.err
}

type fakeReportGenerator struct {
	data models.Data
	err  error
}

func (f fakeReportGenerator) Generate(_ context.Context, _ models.Report) (models.Data, error) {
	return f.data, f.err
}

func TestGenerateReport_NotFoundFromDB(t *testing.T) {
	r := NewReport(fakeReportDB{err: models.ErrNotFound}, fakeReportGenerator{}, slog.New(slog.DiscardHandler))

	_, err := r.GenerateReport("some-id")
	if err != errorz.ErrNotFound {
		t.Fatalf("GenerateReport() error = %v, want errorz.ErrNotFound", err)
	}
}

func TestGenerateReport_NotFoundFromGenerator(t *testing.T) {
	r := NewReport(
		fakeReportDB{report: &models.Report{Name: "r1"}},
		fakeReportGenerator{err: models.ErrNotFound},
		slog.New(slog.DiscardHandler),
	)

	_, err := r.GenerateReport("some-id")
	if err != errorz.ErrNotFound {
		t.Fatalf("GenerateReport() error = %v, want errorz.ErrNotFound", err)
	}
}

func TestGenerateReport_ReturnsGeneratedData(t *testing.T) {
	want := models.Data{FileName: "out.csv"}
	r := NewReport(
		fakeReportDB{report: &models.Report{Name: "r1"}},
		fakeReportGenerator{data: want},
		slog.New(slog.DiscardHandler),
	)

	got, err := r.GenerateReport("some-id")
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	if got.FileName != want.FileName {
		t.Fatalf("GenerateReport() = %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 4: Run the new test to verify it fails, then passes**

Run: `go test ./internal/service/... -run TestGenerateReport_ -v`
Expected before Step 2's fix: `TestGenerateReport_NotFoundFromGenerator` FAILs (`error = InternalError: ..., want errorz.ErrNotFound`).
After Step 2's fix, re-run the same command.
Expected: PASS

- [ ] **Step 5: Update `internal/app/app.go`**

Change the channel declaration (was `reportChan := make(chan models.Report, channelBufferSize)`):

```go
	reportChan := make(chan generator.Job, channelBufferSize)
```

After the existing `gen := generator.New(reportChan, clct, *snd, *delRepo, proc, eval, 4, log)` line, construct the on-demand service (`reportRepo` already exists a few lines below for the tg_bot repository — add a second repository instance for the `internal/repository` package, which is what implements `GetPublicReportByID`):

```go
	gen := generator.New(reportChan, clct, *snd, *delRepo, proc, eval, 4, log)

	reportDBRepo := repository.NewRepository(rdb.GetConn(), log)
	reportGenSvc := service.NewReport(reportDBRepo, generator.ReportGeneratorAdapter{Gen: gen}, log)
```

Add the import (the package is already imported for `repository.NewChatRepository`/`repository.NewUserRepository`/`repository.NewReportRepository` — `repository.NewRepository` lives in the same `support_bot/internal/repository` package, no new import needed). `service` is already imported too.

`reportGenSvc` is consumed by Task 4 (HTTP handler wiring) — store it on the `app` struct for Task 4 to use:

```go
type app struct {
	ctx    context.Context
	cancel context.CancelFunc

	log     *slog.Logger
	storage *postgres.DB
	cfg     *config.Config
	report  *reportApp

	tgBot *telegramBot
	smb   *smb.SMB

	reportGenSvc *service.Report
}
```

and in `init`, after constructing it:

```go
	a.reportGenSvc = reportGenSvc
```

- [ ] **Step 6: Full build**

Run: `go build ./...`
Expected: builds clean (this is the point where the whole module compiles again after the `chan models.Report` → `chan generator.Job` change).

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: PASS (no pre-existing test touches `orchestrator.New`/`ReportC`/`generator.New`, per the earlier `grep -rn` scope check — only `internal/app/app.go`, `internal/generator/generator.go`, `internal/orchestrator/orchestrator.go` reference these).

- [ ] **Step 8: Commit**

```bash
git add internal/orchestrator/orchestrator.go internal/service/report_generator.go internal/service/report_generator_test.go internal/app/app.go
git commit -m "feat: route scheduled and on-demand report generation through the same pool"
```

---

### Task 4: Make the endpoint reachable (HTTP server bring-up)

This task is not about the generation pipeline — it closes a pre-existing gap found during investigation: nothing in `app.go` ever constructs or starts `internal/api/http.Server`, so `GET /api/v1/public/report/{report_id}` cannot be hit no matter what Tasks 1-3 do. Skip this task only if you already have separate plans to bring up the HTTP server.

**Files:**
- Modify: `internal/api/http/server.go` (expose the router)
- Modify: `internal/api/http/handlers/handler.go` (add constructor)
- Modify: `internal/config/config.go` (add `HTTP` field + validation)
- Modify: `internal/config/default.go` (add HTTP defaults)
- Modify: `config/config.example.yaml` (document `http:` section)
- Modify: `internal/app/app.go` (build server, register routes, start/stop lifecycle)

**Interfaces:**
- Consumes: `a.reportGenSvc *service.Report` from Task 3 (satisfies `handlers.ReportProvider` — `GenerateReport(reportID string) (models.Data, error)`).
- Produces: a running `*http.Server` reachable at `cfg.HTTP.Addr()`, stopped from `app.close`.

- [ ] **Step 1: Expose the router on `Server`**

In `internal/api/http/server.go`, add after `Addr()`:

```go
// Router returns the underlying router so callers can register routes
// after construction, once all handler dependencies are built.
func (s *Server) Router() *httplib.Router {
	return s.router
}
```

(`httplib` is already imported in this file.)

- [ ] **Step 2: Add `NewHandler`**

In `internal/api/http/handlers/handler.go`, add:

```go
func NewHandler(rp ReportProvider, log *slog.Logger) *Handler {
	return &Handler{
		rp:  rp,
		log: log.With(slog.Any("module", "http_handler")),
	}
}
```

- [ ] **Step 3: Add the `HTTP` field to the root config**

In `internal/config/config.go`, add the import (aliased — the package is named `http`, colliding with nothing here since `net/http` isn't imported in this file, but the alias keeps the intent obvious):

```go
	apihttp "support_bot/internal/api/http"
```

Add the field to `Config`:

```go
type Config struct {
	Log            logger.LogConfig  `yaml:"log"             comment:"Настройки логгирования"`
	MetabaseDomain string            `yaml:"metabase_domain" comment:"Адрес Metabase для забора данных"                                                                                                                                  env:"METABASE_DOMAIN"`
	AppMetrica     appmetrica.Config `yaml:"appmetrica"                                                                                                                                                                                  env:"APP_METRICA"`
	Jira           jira.Config       `yaml:"jira"                                                                                                                                                                                        env:"JIRA"`
	Lua            lua.Config        `yaml:"lua"             comment:"Настройки Lua-процессора."                                                                                                                                         env:"LUA"`
	Database       postgres.Config   `yaml:"database"        comment:"Настройки подключения к Postgres"`
	TgBot          tgbot.Config      `yaml:"telegram"        comment:"Настройки Telegram-бота.\nИспользуется для приема команд и отправки уведомлений."`
	Timeout        timeout           `yaml:"timeout"         comment:"Настройка таймаутов"`
	SMB            smb.Config        `yaml:"smb"             comment:"Настройки подключения к SMB (Samba) файловой шаре.\nИспользуется для чтения и/или записи файлов на сетевой ресурс.\nПоддерживается аутентификация по логину/паролю."`
	SMTP           smtp.Config       `yaml:"smtp"            comment:"Настройки SMTP-сервера.\nИспользуется для отправки email-уведомлений и отчетов.\nПоддерживается аутентификация по логину и паролю."`
	MaxBot         maxbot.Config     `yaml:"max"             comment:"Настройка Max бота"`
	HTTP           apihttp.Config    `yaml:"http"             comment:"Настройки публичного HTTP API."`
}
```

Fold `HTTP.Validate()` into `Config.Validate()` (add the `"errors"` import):

```go
func (c Config) Validate() error {
	// TODO: add full config validation.
	return errors.Join(c.Log.Validate(), c.HTTP.Validate())
}
```

- [ ] **Step 4: Add HTTP defaults**

In `internal/config/default.go`, add the import (`apihttp "support_bot/internal/api/http"`) and, inside the returned `&Config{...}` literal, add:

```go
		HTTP: apihttp.Config{
			Host:              "127.0.0.1",
			Port:              8080,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
			MaxBodyBytes:      10 << 20,
			AuthToken:         "changeme-auth-token",
		},
```

(`ReadHeaderTimeout` and `MaxHeaderBytes` have no `env`/`yaml` struct tags in `internal/api/http/config.go`, so `cleanenv` never fills them from a config file or environment — they must be set here for the running server to have working timeouts.)

- [ ] **Step 5: Document the new section in `config/config.example.yaml`**

Read the file first to match its existing per-section comment style, then append an `http:` block mirroring the `smb`/`smtp` sections already there, with keys `host`, `port`, `read_timeout`, `write_timeout`, `idle_timeout`, `max_body_bytes`, `auth_token` (matching the `yaml:"..."` tags in `internal/api/http/config.go`).

- [ ] **Step 6: Wire the server into `app.go`**

Add the import (aliased, same reasoning as Step 3): `apihttp "support_bot/internal/api/http"`, plus `"support_bot/internal/api/http/handlers"` and `"support_bot/internal/pkg/httplib"`.

Add an `http *apihttp.Server` field to the `app` struct (alongside the `reportGenSvc` field added in Task 3):

```go
type app struct {
	// ...existing fields...
	reportGenSvc *service.Report
	http         *apihttp.Server
}
```

In `init`, after `a.reportGenSvc = reportGenSvc`:

```go
	httpSrv := apihttp.New(&cfg.HTTP, log)

	reportHandler := handlers.NewHandler(reportGenSvc, log)
	httpSrv.Router().Group("/api/v1/public", func(r *httplib.Router) {
		r.Get("/report/{report_id}", reportHandler.GetGeneratedReportByID)
	})

	a.http = httpSrv
```

In `Start` (currently `a.tgBot.start(); return a.report.start(a.ctx)`), start the HTTP server too:

```go
func (a *app) Start(_ context.Context) error {
	a.tgBot.start()
	a.http.Start()

	return a.report.start(a.ctx)
}
```

In `close` (alongside the existing `a.tgBot`/`a.report` shutdown), stop it:

```go
	if a.http != nil {
		if err := a.http.Shutdown(ctx); err != nil {
			err = errors.Join(err, a.http.Shutdown(ctx))
		}
	}
```

Use the existing `err error` variable already declared later in `close` for `a.smb`/`a.storage` — move `a.http.Shutdown(ctx)` into that same `errors.Join` chain rather than declaring a second one:

```go
	var err error

	if a.http != nil {
		err = errors.Join(err, a.http.Shutdown(ctx))
	}

	if a.smb != nil {
		err = errors.Join(err, a.smb.Close())
	}

	if a.storage != nil {
		err = errors.Join(err, a.storage.Stop(ctx))
	}

	return err
```

- [ ] **Step 7: Full build**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 8: Manual smoke check**

Run the app against a local Postgres with at least one report row that has `access_from_lk = true` and a row in `public_reports` mapping a UUID to it (this plan assumes that table and its population already exist from earlier work — if not, that's out of scope here), then:

```bash
curl -i -H "Authorization: Bearer changeme-auth-token" \
  http://127.0.0.1:8080/api/v1/public/report/<the-public-uuid>
```

Expected: `200 OK` with the exported file body, or `404` for an unknown/unevaluated report, matching the behavior locked in by Tasks 2-3's tests.

- [ ] **Step 9: Commit**

```bash
git add internal/api/http/server.go internal/api/http/handlers/handler.go internal/config/config.go internal/config/default.go config/config.example.yaml internal/app/app.go
git commit -m "feat(http): wire the public report endpoint into app startup"
```

---

## Self-Review

**Spec coverage:**
- Same generation rules for on-demand reports → Task 1 (shared `generate`) + Task 2 (`generateOnDemand` calls the identical `generate`).
- Same limits (bounded concurrency) → Task 2 (`GenerateOnDemand` enqueues onto `Generator.c`, consumed by the existing `numWorkers`-sized pool) + Task 3 (orchestrator and scheduled jobs share that exact channel).
- No recipient delivery for on-demand → Task 2 (`generateOnDemand`/`worker`'s `j.Result != nil` branch never calls `deliver`).
- Negative evaluation → 404 → Task 2 test + Task 3 Step 2 fix + test.
- Endpoint actually reachable → Task 4.

**Placeholder scan:** none — every step has runnable code or an exact shell command.

**Type consistency:** `Job`/`JobResult` (Task 2) are used identically in Task 3's `orchestrator.go` edits and Task 4 doesn't touch them. `service.NewReport(db ReportDB, gen ReportGenerator, log *slog.Logger) *Report` and `ReportGeneratorAdapter.Generate(ctx, report) (models.Data, error)` match the pre-existing `internal/service/report_generator.go` interfaces verbatim — verified against the current file on disk before writing this plan.
