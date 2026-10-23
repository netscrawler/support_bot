package generator

import (
	"context"
	"log/slog"
	"strings"
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

func (f fakeEvaluator) EvalStr(_ context.Context, expr string, _, _ time.Time) (string, error) {
	return expr, nil
}

//go:fix inline
func strPtr(s string) *string { return new(s) }

// TestGenerate_ExportsOnApprove проверяет generate() напрямую: при
// положительном результате оценки условия (evaluate) собранные данные
// экспортируются в указанный формат.
func TestGenerate_ExportsOnApprove(t *testing.T) {
	g := &Generator{
		clct: fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval: fakeEvaluator{approve: true},
		log:  slog.New(slog.DiscardHandler),
	}

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: new("out")}},
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

// TestGenerate_NegativeEvaluationSkipsExport проверяет, что при
// отрицательном результате оценки условия экспорт не выполняется и
// возвращается approve = false, res = nil.
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

// TestGenerator_Generate_ReturnsExportedFiles проверяет публичный метод
// Generate через воркер-пул: запрос отправляется в канал задач и результат
// генерации (approve = true, экспортированный файл) приходит обратно.
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
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: new("out")}},
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

// TestGenerator_Generate_NegativeEvaluationReturnsApproveFalse проверяет,
// что через воркер-пул при отрицательной оценке условия Generate
// возвращает approve = false и data = nil без ошибки.
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

// TestGenerator_Generate_SharesWorkerPoolAcrossCallers проверяет, что
// несколько одновременных вызовов Generate корректно обслуживаются общим
// пулом воркеров и каждый вызывающий получает свой результат.
func TestGenerator_Generate_SharesWorkerPoolAcrossCallers(t *testing.T) {
	g := &Generator{
		c:          make(chan job),
		clct:       fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval:       fakeEvaluator{approve: true},
		numWorkers: 1,
		log:        slog.New(slog.DiscardHandler),
	}

	ctx := t.Context()

	g.Start(ctx)

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports:    []models.Export{{Format: models.ReportFormatCsv, FileName: new("out")}},
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

// TestGenerate_ExportErrors проверяет обработку ошибок экспорта: упали все —
// ошибка с обеими причинами, упал один — остальные доставляются, нет
// экспортов — не ошибка.
func TestGenerate_ExportErrors(t *testing.T) {
	bad1 := models.Export{Format: "bad1"}
	bad2 := models.Export{Format: "bad2"}
	good := models.Export{Format: models.ReportFormatCsv, FileName: new("out")}

	tests := []struct {
		name     string
		exports  []models.Export
		wantRes  int
		wantErrs []string
	}{
		{
			"all fail",
			[]models.Export{bad1, bad2},
			0,
			[]string{"all exports failed", "bad1", "bad2"},
		},
		{"one fails", []models.Export{bad1, good}, 1, nil},
		{"no exports", nil, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				clct: fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
				eval: fakeEvaluator{approve: true},
				log:  slog.New(slog.DiscardHandler),
			}

			_, res, approve, err := g.generate(
				context.Background(),
				models.Report{Name: "r1", Evaluation: "true", Exports: tt.exports},
			)

			if !approve {
				t.Fatalf("approve = false, want true")
			}

			if len(res) != tt.wantRes {
				t.Fatalf("len(res) = %d, want %d", len(res), tt.wantRes)
			}

			if len(tt.wantErrs) == 0 && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}

			for _, w := range tt.wantErrs {
				if err == nil || !strings.Contains(err.Error(), w) {
					t.Fatalf("err = %v, want it to contain %q", err, w)
				}
			}
		})
	}
}

// TestGenerate_IgnoreEvaluateExportsOnNegative проверяет пробный запуск:
// при ложном условии и IgnoreEvaluate экспорт выполняется, approve остаётся false.
func TestGenerate_IgnoreEvaluateExportsOnNegative(t *testing.T) {
	g := &Generator{
		clct: fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
		eval: fakeEvaluator{approve: false},
		log:  slog.New(slog.DiscardHandler),
	}

	report := models.Report{
		Name:           "r1",
		Evaluation:     "false",
		IgnoreEvaluate: true,
		Exports:        []models.Export{{Format: models.ReportFormatCsv, FileName: new("out")}},
	}

	_, res, approve, err := g.generate(context.Background(), report)
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}

	if approve {
		t.Fatalf("generate() approve = true, want false")
	}

	if len(res) != 1 {
		t.Fatalf("generate() res = %+v, want one exported file", res)
	}
}

type rangeEvaluator struct {
	fakeEvaluator
	from, to *time.Time
}

func (r rangeEvaluator) EvalStr(
	_ context.Context,
	expr string,
	from, to time.Time,
) (string, error) {
	*r.from, *r.to = from, to

	return expr, nil
}

// Period запуска важнее DefaultPeriod отчёта; без обоих — DefaultPeriod.
func TestGenerate_PeriodSelection(t *testing.T) {
	cases := []struct {
		name         string
		period, def  models.PeriodName
		wantDuration time.Duration
	}{
		{"default yesterday", "", models.PeriodYesterday, 24 * time.Hour},
		{"run period overrides", models.PeriodWeek, models.PeriodYesterday, 7 * 24 * time.Hour},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var from, to time.Time

			g := &Generator{
				clct: fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
				eval: rangeEvaluator{from: &from, to: &to},
				log:  slog.New(slog.DiscardHandler),
			}

			report := models.Report{
				Name:          "r1",
				Period:        c.period,
				DefaultPeriod: c.def,
				Queries: []models.Card{
					{Title: "q1", RawParams: []byte(`{"a":"period_from"}`)},
				},
			}

			if _, _, _, err := g.generate(context.Background(), report); err != nil {
				t.Fatalf("generate() error = %v", err)
			}

			// Сутки в локали могут быть 23/25 ч при смене времени; сверяем по дням.
			if days := to.Sub(from).Round(24 * time.Hour); days != c.wantDuration {
				t.Errorf("range = %v..%v, want %v long", from, to, c.wantDuration)
			}
		})
	}
}
