package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"support_bot/internal/delivery"
	"support_bot/internal/generator"
	"support_bot/internal/models"
	"support_bot/internal/pkg/secret"
)

type fakeCollector struct {
	data models.Dataset
	err  error
}

func (f fakeCollector) Collect(
	_ context.Context,
	_ ...models.Card,
) (models.Dataset, error) {
	return f.data, f.err
}

type fakeEvaluator struct {
	approve bool
}

func (f fakeEvaluator) Evaluate(
	_ context.Context,
	_ models.Dataset,
	_ string,
) (bool, error) {
	return f.approve, nil
}

func (f fakeEvaluator) EvalStr(
	_ context.Context,
	expr string,
	_, _ time.Time,
) (string, error) {
	return expr, nil
}

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

// TestOrchestrator_Generate_ReturnsFirstExport проверяет, что при
// положительной оценке условия Orchestrator.Generate возвращает
// экспортированный файл, полученный от generator.Generator.
func TestOrchestrator_Generate_ReturnsFirstExport(t *testing.T) {
	o := &Orchestrator{
		gen: newTestGenerator(t, true),
		log: slog.New(slog.DiscardHandler),
	}

	report := models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports: []models.Export{
			{Format: models.ReportFormatCsv, FileName: new("out")},
		},
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

// TestOrchestrator_Generate_NegativeEvaluationIsNotFound проверяет, что при
// отрицательной оценке условия Orchestrator.Generate транслирует это в
// models.ErrNotFound (отчет не подлежит отправке).
func TestOrchestrator_Generate_NegativeEvaluationIsNotFound(t *testing.T) {
	o := &Orchestrator{
		gen: newTestGenerator(t, false),
		log: slog.New(slog.DiscardHandler),
	}

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

func (f fakeLoader) LoadByEvent(
	_ context.Context,
	_ string,
	_ bool,
) (*models.Report, error) {
	r := f.report

	return &r, nil
}

// blockingCollector.Collect блокируется до закрытия release, отслеживая
// число одновременных вызовов (и пиковое значение), чтобы тест мог проверить
// границы конкурентности без гонки по времени.
type blockingCollector struct {
	release  chan struct{}
	inFlight *int32
	peak     *int32
}

func (b blockingCollector) Collect(
	_ context.Context,
	_ ...models.Card,
) (models.Dataset, error) {
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

// TestOrchestrator_ProcessGenReportEvent_BoundsInFlightGenerations защищает
// от утечки горутин на каждое событие: processGenReportEvent никогда не
// должен допускать больше maxInFlightGenerations одновременных вызовов
// generateAndDeliver, сколько бы событий ни пришло параллельно.
func TestOrchestrator_ProcessGenReportEvent_BoundsInFlightGenerations(
	t *testing.T,
) {
	var inFlight, peak int32

	release := make(chan struct{})
	clct := blockingCollector{
		release:  release,
		inFlight: &inFlight,
		peak:     &peak,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	g := generator.New(
		clct,
		nil,
		fakeEvaluator{approve: false},
		maxInFlightGenerations*2,
		slog.New(slog.DiscardHandler),
	)
	g.Start(ctx)

	o := &Orchestrator{
		gen: g,
		rL: fakeLoader{
			report: models.Report{Name: "r1", Evaluation: "false"},
		},
		runs:   &fakeRecorder{},
		genSem: make(chan struct{}, maxInFlightGenerations),
		log:    slog.New(slog.DiscardHandler),
	}

	var wg sync.WaitGroup

	for range maxInFlightGenerations * 4 {
		wg.Go(func() {
			o.processGenReportEvent(
				ctx,
				"ev",
				true,
				nil,
				models.RunKindCron,
				"",
			)
		})
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
	wg.Wait() // доказывает лишь, что каждый вызов processGenReportEvent завершился, т.е. прошёл семафор

	deadline = time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&inFlight) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for in-flight generations to drain, got %d",
				atomic.LoadInt32(&inFlight),
			)
		}

		time.Sleep(5 * time.Millisecond)
	}

	if p := atomic.LoadInt32(&peak); p > maxInFlightGenerations {
		t.Fatalf(
			"peak in-flight generations = %d, want <= %d",
			p,
			maxInFlightGenerations,
		)
	}
}

// fakeRecorder сохраняет записи журнала запусков.
type fakeRecorder struct {
	mu   sync.Mutex
	runs []models.Run

	prevHash  string // ответ LastSuccessHash
	prevOK    bool
	hashCalls int
}

func (f *fakeRecorder) LastSuccessHash(
	_ context.Context,
	_ string,
) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.hashCalls++

	return f.prevHash, f.prevOK
}

func (f *fakeRecorder) Record(_ context.Context, run models.Run) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.runs = append(f.runs, run)
}

// wait ждёт запись (генерация идёт в горутине) и возвращает копию записей.
func (f *fakeRecorder) wait(t *testing.T) []models.Run {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		f.mu.Lock()

		got := append([]models.Run(nil), f.runs...)
		f.mu.Unlock()

		if len(got) >= 1 {
			return got
		}

		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for a run record, got %d",
				len(got),
			)
		}

		time.Sleep(5 * time.Millisecond)
	}
}

// fakeSmb падает на пути "bad"; остальные Upload успешны.
type fakeSmb struct{}

func (fakeSmb) Upload(_ context.Context, remote string, _ models.Data) error {
	if remote == "bad" {
		return errors.New("upload failed")
	}

	return nil
}

type fakeSaver struct{}

func (fakeSaver) SaveTgMsg(
	context.Context,
	string,
	[]models.SentMessage,
) error {
	return nil
}

// fakeTg записывает тексты SendText.
type fakeTg struct {
	delivery.TgSender

	mu    sync.Mutex
	texts []string
}

func (f *fakeTg) SendText(
	_ context.Context,
	_ models.TgChat,
	s string,
) (*models.SentMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.texts = append(f.texts, s)

	return &models.SentMessage{}, nil
}

func (f *fakeTg) got() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.texts...)
}

func smbRcpt(path string) models.Recipient {
	return models.Recipient{Type: models.SambaRecipient, RemotePath: &path}
}

func goodReport(rcpts ...models.Recipient) models.Report {
	return models.Report{
		Name:       "r1",
		Evaluation: "true",
		Exports: []models.Export{
			{Format: models.ReportFormatCsv, FileName: new("out")},
		},
		Recipients: rcpts,
	}
}

func newRunOrch(
	t *testing.T,
	coll fakeCollector,
	approve bool,
) (*Orchestrator, *fakeRecorder) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	g := generator.New(
		coll,
		nil,
		fakeEvaluator{approve: approve},
		1,
		slog.New(slog.DiscardHandler),
	)
	g.Start(ctx)

	rec := &fakeRecorder{}

	return &Orchestrator{
		gen:         g,
		snd:         delivery.NewDispatcher(&fakeTg{}, fakeSmb{}, nil, nil),
		sentMsgRepo: fakeSaver{},
		runs:        rec,
		genSem:      make(chan struct{}, maxInFlightGenerations),
		log:         slog.New(slog.DiscardHandler),
	}, rec
}

// TestOrchestrator_GenerateAndDeliver_RecordsRun проверяет статус записи
// журнала для каждого исхода запуска.
func TestOrchestrator_GenerateAndDeliver_RecordsRun(t *testing.T) {
	data := models.Dataset{"q1": {{"col": "val"}}}
	badExport := goodReport(smbRcpt("ok"))

	badExport.Exports = []models.Export{{Format: "nope"}}

	tests := []struct {
		name       string
		coll       fakeCollector
		approve    bool
		report     models.Report
		wantStatus models.RunStatus
		wantReason string
		wantErr    string
	}{
		{
			"success",
			fakeCollector{data: data},
			true,
			goodReport(smbRcpt("ok")),
			models.RunStatusSuccess,
			"",
			"",
		},
		{
			"skipped",
			fakeCollector{data: data},
			false,
			goodReport(smbRcpt("ok")),
			models.RunStatusSkipped,
			"evaluate",
			"",
		},
		{
			"generate error",
			fakeCollector{err: errors.New("db down")},
			true,
			goodReport(smbRcpt("ok")),
			models.RunStatusFailed,
			"",
			"db down",
		},
		{
			"all exports failed",
			fakeCollector{data: data},
			true,
			badExport,
			models.RunStatusFailed,
			"",
			"all exports failed",
		},
		{
			"no recipients",
			fakeCollector{data: data},
			true,
			goodReport(),
			models.RunStatusFailed,
			"",
			"empty targets list",
		},
		{
			"all sends failed",
			fakeCollector{data: data},
			true,
			goodReport(smbRcpt("bad")),
			models.RunStatusFailed,
			"",
			"upload failed",
		},
		{
			"partial",
			fakeCollector{data: data},
			true,
			goodReport(smbRcpt("ok"), smbRcpt("bad")),
			models.RunStatusPartial,
			"",
			"upload failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, rec := newRunOrch(t, tt.coll, tt.approve)

			chatID := int64(7)
			o.generateAndDeliver(
				context.Background(),
				tt.report,
				models.RunKindManual,
				&chatID,
			)

			runs := rec.wait(t)
			if len(runs) != 1 {
				t.Fatalf("got %d records, want exactly 1", len(runs))
			}

			r := runs[0]
			if r.Status != tt.wantStatus || r.Reason != tt.wantReason {
				t.Fatalf(
					"status/reason = %s/%q, want %s/%q",
					r.Status,
					r.Reason,
					tt.wantStatus,
					tt.wantReason,
				)
			}

			if tt.wantErr == "" && r.Error != "" ||
				!strings.Contains(r.Error, tt.wantErr) {
				t.Fatalf(
					"error = %q, want it to contain %q",
					r.Error,
					tt.wantErr,
				)
			}

			if r.ReportName != "r1" || r.Kind != models.RunKindManual ||
				r.ChatID != &chatID {
				t.Fatalf("unexpected run fields: %+v", r)
			}

			if r.StartedAt.IsZero() || r.FinishedAt.Before(r.StartedAt) {
				t.Fatalf("bad timestamps: %+v", r)
			}
		})
	}
}

// TestOrchestrator_GenerateAndDeliver_Dedup проверяет пропуск дубля только для cron
// с report.Dedup и запись DataHash.
func TestOrchestrator_GenerateAndDeliver_Dedup(t *testing.T) {
	data := models.Dataset{"q1": {{"col": "val"}}}
	same := data.Hash()

	tests := []struct {
		name       string
		dedup      bool
		kind       models.RunKind
		prevHash   string
		prevOK     bool
		wantStatus models.RunStatus
		wantReason string
		wantHash   string
		wantCalls  int
	}{
		{
			"cron same hash", true, models.RunKindCron, same, true,
			models.RunStatusSkipped, "duplicate", same, 1,
		},
		{
			"cron other hash", true, models.RunKindCron, "other", true,
			models.RunStatusSuccess, "", same, 1,
		},
		{
			"cron no previous", true, models.RunKindCron, "", false,
			models.RunStatusSuccess, "", same, 1,
		},
		{
			"manual same hash", true, models.RunKindManual, same, true,
			models.RunStatusSuccess, "", same, 0,
		},
		{
			"dedup off", false, models.RunKindCron, same, true,
			models.RunStatusSuccess, "", same, 0,
		},
		{
			"lk no hash", true, models.RunKindLK, same, true,
			models.RunStatusSuccess, "", "", 0,
		},
		{
			"test no hash", true, models.RunKindTest, same, true,
			models.RunStatusSuccess, "", "", 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, rec := newRunOrch(t, fakeCollector{data: data}, true)

			rec.prevHash, rec.prevOK = tt.prevHash, tt.prevOK

			report := goodReport(smbRcpt("ok"))

			report.Dedup = tt.dedup

			o.generateAndDeliver(context.Background(), report, tt.kind, nil)

			r := rec.wait(t)[0]
			if r.Status != tt.wantStatus || r.Reason != tt.wantReason {
				t.Fatalf("status/reason = %s/%q, want %s/%q",
					r.Status, r.Reason, tt.wantStatus, tt.wantReason)
			}

			if r.DataHash != tt.wantHash {
				t.Fatalf("DataHash = %q, want %q", r.DataHash, tt.wantHash)
			}

			rec.mu.Lock()

			calls := rec.hashCalls
			rec.mu.Unlock()

			if calls != tt.wantCalls {
				t.Fatalf(
					"LastSuccessHash calls = %d, want %d",
					calls,
					tt.wantCalls,
				)
			}
		})
	}
}

// TestOrchestrator_Run_PassesKindAndChat проверяет, что run() сопоставляет
// каналы событий типам запуска, а для ЛК пробрасывает ChatID получателя.
func TestOrchestrator_Run_PassesKindAndChat(t *testing.T) {
	rcpt := smbRcpt("ok")

	rcpt.Chat = &models.Chat{ChatID: 42}

	tests := []struct {
		name     string
		send     func(o *Orchestrator)
		wantKind models.RunKind
		wantChat *int64
	}{
		{
			"cron",
			func(o *Orchestrator) { o.EventC <- models.Event{Name: "ev"} },
			models.RunKindCron,
			nil,
		},
		{"manual", func(o *Orchestrator) {
			o.SpecialEventC <- models.SpecialEventForLK{Event: models.Event{Name: "ev", Type: models.EventTypeGenReport}}
		}, models.RunKindManual, nil},
		{"test", func(o *Orchestrator) {
			o.SpecialEventC <- models.SpecialEventForLK{
				Event:     models.Event{Name: "ev", Type: models.EventTypeGenReportForTG},
				Recipient: rcpt,
				Test:      true,
			}
		}, models.RunKindTest, new(int64(42))},
		{"lk", func(o *Orchestrator) {
			o.SpecialEventC <- models.SpecialEventForLK{
				Event:     models.Event{Name: "ev", Type: models.EventTypeGenReportForTG},
				Recipient: rcpt,
			}
		}, models.RunKindLK, new(int64(42))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, rec := newRunOrch(
				t,
				fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
				true,
			)

			o.EventC = make(chan models.Event)
			o.SpecialEventC = make(chan models.SpecialEventForLK)
			o.rL = fakeLoader{report: goodReport(smbRcpt("ok"))}

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			go o.run(ctx)

			tt.send(o)

			r := rec.wait(t)[0]
			if r.Kind != tt.wantKind {
				t.Fatalf("kind = %s, want %s", r.Kind, tt.wantKind)
			}

			if (r.ChatID == nil) != (tt.wantChat == nil) ||
				r.ChatID != nil && *r.ChatID != *tt.wantChat {
				t.Fatalf("chat id = %v, want %v", r.ChatID, tt.wantChat)
			}
		})
	}
}

// TestOrchestrator_GenerateAndDeliver_TestKind проверяет пробный запуск админа:
// доставка при ложном evaluate, служебные строки и их отсутствие для lk.
func TestOrchestrator_GenerateAndDeliver_TestKind(t *testing.T) {
	data := models.Dataset{"q1": {{"col": "val"}}}
	chatID := int64(7)

	tests := []struct {
		name       string
		coll       fakeCollector
		kind       models.RunKind
		wantStatus models.RunStatus
		wantReason string
		wantNote   string // "" — служебных строк быть не должно
		wantEscape string
	}{
		{
			"test evaluate false",
			fakeCollector{data: data},
			models.RunKindTest,
			models.RunStatusSuccess, "evaluate_false", "evaluate = false", "",
		},
		{
			"test collector error",
			fakeCollector{err: errors.New("bad <b>")},
			models.RunKindTest,
			models.RunStatusFailed, "", "❌", "&lt;b&gt;",
		},
		{
			"lk evaluate false",
			fakeCollector{data: data},
			models.RunKindLK,
			models.RunStatusSkipped, "evaluate", "", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, rec := newRunOrch(t, tt.coll, false)
			tg := &fakeTg{}

			o.snd = delivery.NewDispatcher(tg, fakeSmb{}, nil, nil)

			report := goodReport(smbRcpt("ok"))

			report.IgnoreEvaluate = tt.kind == models.RunKindTest

			o.generateAndDeliver(context.Background(), report, tt.kind, &chatID)

			r := rec.wait(t)[0]
			if r.Kind != tt.kind || r.Status != tt.wantStatus ||
				r.Reason != tt.wantReason {
				t.Fatalf("run = %s/%s/%q", r.Kind, r.Status, r.Reason)
			}

			texts := tg.got()
			if tt.wantNote == "" {
				if len(texts) != 0 {
					t.Fatalf("unexpected notes: %v", texts)
				}

				return
			}

			if len(texts) != 1 || !strings.Contains(texts[0], tt.wantNote) ||
				!strings.Contains(texts[0], tt.wantEscape) {
				t.Fatalf(
					"notes = %q, want %q / %q",
					texts,
					tt.wantNote,
					tt.wantEscape,
				)
			}

			if tt.wantNote == "❌" && !strings.HasPrefix(texts[0], "❌") {
				t.Fatalf("note = %q, want ❌ prefix", texts[0])
			}
		})
	}
}

// Ошибка пробного запуска уходит админу без секретов.
func TestOrchestrator_TestKind_RedactsSecretInError(t *testing.T) {
	secret.Register("orch-secret-777")

	chatID := int64(7)
	o, _ := newRunOrch(
		t,
		fakeCollector{err: errors.New("GET /botorch-secret-777/x: boom")},
		true,
	)
	tg := &fakeTg{}

	o.snd = delivery.NewDispatcher(tg, fakeSmb{}, nil, nil)
	o.generateAndDeliver(
		context.Background(),
		goodReport(smbRcpt("ok")),
		models.RunKindTest,
		&chatID,
	)

	texts := tg.got()
	if len(texts) != 1 || strings.Contains(texts[0], "orch-secret-777") ||
		!strings.Contains(texts[0], "***") {
		t.Fatalf("notes = %q, want masked secret", texts)
	}
}

type spanEvaluator struct {
	fakeEvaluator

	span chan time.Duration
}

func (s spanEvaluator) EvalStr(
	_ context.Context,
	expr string,
	from, to time.Time,
) (string, error) {
	s.span <- to.Sub(from)

	return expr, nil
}

// Период события берётся только для lk; cron/manual идут с периодом по умолчанию (today).
func TestOrchestrator_ProcessGenReportEvent_Period(t *testing.T) {
	cases := []struct {
		name   string
		kind   models.RunKind
		period models.PeriodName
		want   time.Duration
	}{
		{
			"lk берёт период события",
			models.RunKindLK,
			models.PeriodWeek,
			7 * 24 * time.Hour,
		},
		{"cron игнорирует период", models.RunKindCron, "", 24 * time.Hour},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			span := make(chan time.Duration, 10)
			g := generator.New(
				fakeCollector{data: models.Dataset{"q1": {{"col": "val"}}}},
				nil,
				spanEvaluator{span: span},
				1,
				slog.New(slog.DiscardHandler),
			)
			g.Start(ctx)

			report := models.Report{
				Name: "r1",
				Queries: []models.Card{
					{Title: "q1", RawParams: []byte(`{"a":"period_from"}`)},
				},
			}
			o := &Orchestrator{
				gen:    g,
				rL:     fakeLoader{report: report},
				runs:   &fakeRecorder{},
				genSem: make(chan struct{}, maxInFlightGenerations),
				log:    slog.New(slog.DiscardHandler),
			}

			o.processGenReportEvent(ctx, "ev", false, nil, c.kind, c.period)

			select {
			case got := <-span:
				// Сутки в локали могут быть 23/25 ч при смене времени.
				if got.Round(24*time.Hour) != c.want {
					t.Errorf("range length = %v, want about %v", got, c.want)
				}

			case <-time.After(2 * time.Second):
				t.Fatal("EvalStr not called")
			}
		})
	}
}
