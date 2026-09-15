package generator

import (
	"context"
	"errors"
	"log/slog"
	"support_bot/internal/models"
	"testing"
	"time"
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

func TestGenerator_Generate_ReturnsExportedFiles(t *testing.T) {
	g := &Generator{
		c:          make(chan job),
		clct:       fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval:       fakeEvaluator{approve: true},
		numWorkers: 1,
		log:        slog.New(slog.DiscardHandler),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	g.Start(ctx)

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: strPtr("out")}},
	}

	_, data, approve, err := g.Generate(ctx, report)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !approve {
		t.Fatalf("Generate() approve = false, want true")
	}

	if len(data) != 1 || data[0].FileName != "out_q1.csv" {
		t.Fatalf("Generate() data = %+v", data)
	}
}

func TestGenerator_Generate_NegativeEvaluationReturnsApproveFalse(t *testing.T) {
	g := &Generator{
		c:          make(chan job),
		clct:       fakeCollector{data: models.Dataset{}},
		eval:       fakeEvaluator{approve: false},
		numWorkers: 1,
		log:        slog.New(slog.DiscardHandler),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	g.Start(ctx)

	_, data, approve, err := g.Generate(ctx, models.Report{Name: "r1", Evaluation: "false"})
	if err != nil {
		t.Fatalf("Generate() error = %v, want nil", err)
	}

	if approve {
		t.Fatalf("Generate() approve = true, want false")
	}

	if data != nil {
		t.Fatalf("Generate() data = %+v, want nil", data)
	}
}

func TestGenerator_Generate_SharesWorkerPoolAcrossCallers(t *testing.T) {
	g := &Generator{
		c:          make(chan job),
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
		data []models.Data
		err  error
	}

	results := make(chan outcome, 2)

	for range 2 {
		go func() {
			_, data, _, err := g.Generate(context.Background(), report)
			results <- outcome{data: data, err: err}
		}()
	}

	for range 2 {
		select {
		case o := <-results:
			if o.err != nil {
				t.Errorf("Generate() error = %v", o.err)
			}

			if len(o.data) != 1 || o.data[0].FileName != "out_q1.csv" {
				t.Errorf("Generate() data = %+v", o.data)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for Generate result")
		}
	}
}

func TestReportGeneratorAdapter_Generate_ReturnsFirstExport(t *testing.T) {
	g := &Generator{
		c:          make(chan job),
		clct:       fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval:       fakeEvaluator{approve: true},
		numWorkers: 1,
		log:        slog.New(slog.DiscardHandler),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	g.Start(ctx)

	adapter := ReportGeneratorAdapter{Gen: g}

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: strPtr("out")}},
	}

	data, err := adapter.Generate(ctx, report)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if data.FileName != "out_q1.csv" {
		t.Fatalf("Generate() = %+v, want FileName out_q1.csv", data)
	}
}

func TestReportGeneratorAdapter_Generate_NegativeEvaluationIsNotFound(t *testing.T) {
	g := &Generator{
		c:          make(chan job),
		clct:       fakeCollector{data: models.Dataset{}},
		eval:       fakeEvaluator{approve: false},
		numWorkers: 1,
		log:        slog.New(slog.DiscardHandler),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	g.Start(ctx)

	adapter := ReportGeneratorAdapter{Gen: g}

	_, err := adapter.Generate(ctx, models.Report{Name: "r1", Evaluation: "false"})
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("Generate() error = %v, want models.ErrNotFound", err)
	}
}
