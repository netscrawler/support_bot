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

func TestReportGeneratorAdapter_Generate_ReturnsFirstExport(t *testing.T) {
	g := &Generator{
		c:          make(chan Job),
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
		c:          make(chan Job),
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
