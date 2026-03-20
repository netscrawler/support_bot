package actions

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"support_bot/internal/core/workflow/registry"

	workflow "support_bot/internal/core/workflow"

	models "support_bot/internal/models/report"
)

type fakeCollector struct {
	called int
	cards  []models.Card
	data   map[string][]map[string]any
	err    error
}

func (f *fakeCollector) Collect(
	_ context.Context,
	cards ...models.Card,
) (map[string][]map[string]any, error) {
	f.called++
	f.cards = cards

	if f.err != nil {
		return nil, f.err
	}

	return f.data, nil
}

type fakeEvaluator struct {
	called int
	data   map[string][]map[string]any
	expr   string
	ok     bool
	err    error
}

func (f *fakeEvaluator) Evaluate(
	_ context.Context,
	data map[string][]map[string]any,
	expr string,
) (bool, error) {
	f.called++
	f.data = data
	f.expr = expr

	if f.err != nil {
		return false, f.err
	}

	return f.ok, nil
}

type fakeSender struct {
	called  int
	targets []models.Targeted
	reports []models.ExportedReport
	err     error
}

func (f *fakeSender) Send(
	_ context.Context,
	targets []models.Targeted,
	reports []models.ExportedReport,
) error {
	f.called++
	f.targets = targets
	f.reports = reports

	return f.err
}

func TestRegisterBuiltins_Success(t *testing.T) {
	reg := registry.New()
	deps := BuiltinDeps{
		Collector: &fakeCollector{},
		Evaluator: &fakeEvaluator{},
		Sender:    &fakeSender{},
	}

	if err := RegisterBuiltins(reg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, typ := range []string{
		workflow.ActionTypeCollect,
		workflow.ActionTypeEvaluate,
		workflow.ActionTypeExport,
		workflow.ActionTypeSend,
	} {
		if !reg.Has(typ) {
			t.Fatalf("expected registry to contain %s", typ)
		}
	}
}

func TestCollectAction_Execute(t *testing.T) {
	fc := &fakeCollector{data: map[string][]map[string]any{"sheet": {{"v": 1}}}}
	a := NewCollectAction(fc)

	raw := json.RawMessage(`{"cards":[{"card_uuid":"id1","title":"sheet"}]}`)
	out, err := a.Execute(context.Background(), registry.ActionInput{Config: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fc.called != 1 {
		t.Fatalf("want one call, got %d", fc.called)
	}

	if len(fc.cards) != 1 || fc.cards[0].CardUUID != "id1" {
		t.Fatalf("unexpected cards: %#v", fc.cards)
	}

	data, ok := out.Data.(map[string][]map[string]any)
	if !ok {
		t.Fatalf("unexpected output type: %T", out.Data)
	}

	if data["sheet"][0]["v"] != 1 {
		t.Fatalf("unexpected output data: %#v", data)
	}
}

func TestEvaluateAction_Execute(t *testing.T) {
	fe := &fakeEvaluator{ok: true}
	a := NewEvaluateAction(fe)

	raw := json.RawMessage(`{"expr":"[*]","data":{"sheet":[{"v":1}]}}`)
	out, err := a.Execute(context.Background(), registry.ActionInput{Config: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fe.called != 1 || fe.expr != "[*]" {
		t.Fatalf("unexpected evaluator call: called=%d expr=%q", fe.called, fe.expr)
	}

	m, ok := out.Data.(map[string]any)
	if !ok {
		t.Fatalf("unexpected output type: %T", out.Data)
	}

	if m["approved"] != true {
		t.Fatalf("unexpected approve flag: %#v", m)
	}
}

func TestSendAction_ApprovedFalse_SkipsSend(t *testing.T) {
	fs := &fakeSender{}
	a := NewSendAction(fs)

	raw := json.RawMessage(`{"approved":false}`)
	out, err := a.Execute(context.Background(), registry.ActionInput{Config: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fs.called != 0 {
		t.Fatalf("sender should not be called, got %d", fs.called)
	}

	m, ok := out.Data.(map[string]any)
	if !ok || m["skipped"] != true {
		t.Fatalf("unexpected output: %#v", out.Data)
	}
}

func TestSendAction_ExportFailure_DoesNotAbort(t *testing.T) {
	fs := &fakeSender{}
	a := NewSendAction(fs)

	raw := json.RawMessage(`{
		"data": {"sheet": [{"v": 1}]},
		"exports": [
			{"format": "invalid", "fileName": "bad.bin"},
			{"format": "csv", "fileName": "ok.csv"}
		],
		"recipients": [{"type": "tg", "chat": {"chatID": 1}}]
	}`)

	out, err := a.Execute(context.Background(), registry.ActionInput{Config: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fs.called != 1 {
		t.Fatalf("sender should be called once, got %d", fs.called)
	}

	if len(fs.reports) != 1 {
		t.Fatalf("expected one successfully exported report, got %d", len(fs.reports))
	}

	m, ok := out.Data.(map[string]any)
	if !ok || m["sent"] != true {
		t.Fatalf("unexpected output: %#v", out.Data)
	}
}

func TestSendAction_TargetResolveError_WithValidTargets_StillSends(t *testing.T) {
	fs := &fakeSender{}
	a := NewSendAction(fs)

	raw := json.RawMessage(`{
		"reports": [{"Msg": "ping"}],
		"recipients": [
			{"type": "email", "email": {"dest": ["ops@example.com"], "subject": "{{"}},
			{"type": "tg", "chat": {"chatID": 11}}
		]
	}`)

	if _, err := a.Execute(context.Background(), registry.ActionInput{Config: raw}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fs.called != 1 {
		t.Fatalf("sender should be called once, got %d", fs.called)
	}

	if len(fs.targets) == 0 {
		t.Fatal("expected at least one resolved target")
	}
}

func TestSendAction_UsesPreparedReports(t *testing.T) {
	fs := &fakeSender{}
	a := NewSendAction(fs)

	raw := json.RawMessage(`{
		"reports": [{"Msg": "hello from export", "Parse": "HTML"}],
		"recipients": [{"type": "tg", "chat": {"chatID": 11}}]
	}`)

	if _, err := a.Execute(context.Background(), registry.ActionInput{Config: raw}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fs.called != 1 {
		t.Fatalf("sender should be called once, got %d", fs.called)
	}

	if len(fs.reports) != 1 {
		t.Fatalf("expected one prepared report, got %d", len(fs.reports))
	}

	textReport, ok := fs.reports[0].(*models.TextData)
	if !ok {
		t.Fatalf("expected *models.TextData, got %T", fs.reports[0])
	}

	if textReport.Msg == "" {
		t.Fatal("expected non-empty message from prepared reports")
	}
}

func TestCollectAction_PropagatesError(t *testing.T) {
	a := NewCollectAction(&fakeCollector{err: errors.New("boom")})
	raw := json.RawMessage(`{"cards":[{"card_uuid":"id1","title":"sheet"}]}`)

	if _, err := a.Execute(context.Background(), registry.ActionInput{Config: raw}); err == nil {
		t.Fatal("want error")
	}
}

func TestQueryAction_Execute_FilterQuery(t *testing.T) {
	a := NewQueryAction()

	raw := json.RawMessage(`{
		"data": {
			"sales": [
				{"product": "A", "amount": 100},
				{"product": "B", "amount": 200},
				{"product": "C", "amount": 50}
			]
		},
		"queries": [
			{"name": "big_sales", "sql": "SELECT product, amount FROM sales WHERE amount > 100"}
		]
	}`)

	out, err := a.Execute(context.Background(), registry.ActionInput{Config: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, ok := out.Data.(map[string][]map[string]any)
	if !ok {
		t.Fatalf("unexpected output type: %T", out.Data)
	}

	rows := results["big_sales"]
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d: %#v", len(rows), rows)
	}

	if rows[0]["product"] != "B" {
		t.Fatalf("unexpected product: %#v", rows[0])
	}
}

func TestQueryAction_Execute_MultipleQueries(t *testing.T) {
	a := NewQueryAction()

	raw := json.RawMessage(`{
		"data": {
			"orders": [
				{"id": 1, "status": "done",    "total": 100},
				{"id": 2, "status": "pending", "total": 200},
				{"id": 3, "status": "done",    "total": 300}
			]
		},
		"queries": [
			{"name": "done",    "sql": "SELECT id, total FROM orders WHERE status = 'done'"},
			{"name": "pending", "sql": "SELECT id, total FROM orders WHERE status = 'pending'"}
		]
	}`)

	out, err := a.Execute(context.Background(), registry.ActionInput{Config: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, ok := out.Data.(map[string][]map[string]any)
	if !ok {
		t.Fatalf("unexpected output type: %T", out.Data)
	}

	if len(results["done"]) != 2 {
		t.Fatalf("want 2 done rows, got %d", len(results["done"]))
	}

	if len(results["pending"]) != 1 {
		t.Fatalf("want 1 pending row, got %d", len(results["pending"]))
	}
}

func TestQueryAction_Execute_NoQueries_ReturnsError(t *testing.T) {
	a := NewQueryAction()
	raw := json.RawMessage(`{"data":{}, "queries":[]}`)

	if _, err := a.Execute(context.Background(), registry.ActionInput{Config: raw}); err == nil {
		t.Fatal("want error for empty queries")
	}
}

func TestQueryAction_Execute_BadSQL_ReturnsError(t *testing.T) {
	a := NewQueryAction()
	raw := json.RawMessage(`{
		"data": {},
		"queries": [{"name": "fail", "sql": "SELECT * FROM non_existent_table"}]
	}`)

	if _, err := a.Execute(context.Background(), registry.ActionInput{Config: raw}); err == nil {
		t.Fatal("want error for bad SQL")
	}
}

func TestRegisterBuiltins_IncludesQuery(t *testing.T) {
	reg := registry.New()
	deps := BuiltinDeps{
		Collector: &fakeCollector{},
		Evaluator: &fakeEvaluator{},
		Sender:    &fakeSender{},
	}

	if err := RegisterBuiltins(reg, deps); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reg.Has(workflow.ActionTypeQuery) {
		t.Fatalf("expected registry to contain %s", workflow.ActionTypeQuery)
	}
}
