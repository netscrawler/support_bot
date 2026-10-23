package runlog

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"support_bot/internal/metrics"
	"support_bot/internal/models"
	"support_bot/internal/pkg/secret"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	mu         sync.Mutex
	runs       []models.Run
	lastCalls  int
	insertErr  error
	hash       string
	hashErr    error
	deleteCall chan time.Time
}

func (s *fakeStore) Insert(_ context.Context, run models.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.insertErr != nil {
		return s.insertErr
	}

	s.runs = append(s.runs, run)

	return nil
}

func (s *fakeStore) LastStatus(
	_ context.Context,
	name string,
) (models.RunStatus, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastCalls++
	for _, r := range slices.Backward(s.runs) {
		if r.ReportName == name &&
			(r.Kind == models.RunKindCron || r.Kind == models.RunKindManual) {
			return r.Status, true, nil
		}
	}

	return "", false, nil
}

func (s *fakeStore) LastSuccessHash(
	_ context.Context,
	_ string,
) (string, bool, error) {
	return s.hash, s.hash != "", s.hashErr
}

func (s *fakeStore) DeleteOlderThan(
	_ context.Context,
	before time.Time,
) (int64, error) {
	s.deleteCall <- before

	return 0, nil
}

type fakeAlerter struct {
	mu    sync.Mutex
	texts []string
}

func (a *fakeAlerter) Alert(_ context.Context, text string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.texts = append(a.texts, text)

	return nil
}

func setup() (*Recorder, *fakeStore, *fakeAlerter) {
	st, al := &fakeStore{deleteCall: make(chan time.Time, 4)}, &fakeAlerter{}

	return New(st, al, Config{}, slog.New(slog.DiscardHandler)), st, al
}

func run(
	kind models.RunKind,
	status models.RunStatus,
	errText string,
) models.Run {
	return models.Run{
		ReportName: "r1",
		Kind:       kind,
		Status:     status,
		Error:      errText,
	}
}

func TestLastSuccessHash(t *testing.T) {
	r, st, _ := setup()

	st.hash = "h"

	h, ok := r.LastSuccessHash(t.Context(), "r1")
	assert.True(t, ok)
	assert.Equal(t, "h", h)

	st.hashErr = errors.New("db down")
	h, ok = r.LastSuccessHash(t.Context(), "r1")
	assert.False(t, ok)
	assert.Empty(t, h)
}

func TestFirstFailureAlerts(t *testing.T) {
	r, _, al := setup()
	r.Record(
		t.Context(),
		run(models.RunKindCron, models.RunStatusFailed, "boom"),
	)
	require.Len(t, al.texts, 1)
	assert.Contains(t, al.texts[0], "<b>r1</b> (cron): failed")
	assert.Contains(t, al.texts[0], "<pre>boom</pre>")
}

func TestRepeatedFailureSilent(t *testing.T) {
	r, _, al := setup()
	r.Record(t.Context(), run(models.RunKindCron, models.RunStatusFailed, "a"))
	r.Record(t.Context(), run(models.RunKindCron, models.RunStatusFailed, "b"))
	assert.Len(t, al.texts, 1)
}

func TestRecovery(t *testing.T) {
	r, _, al := setup()
	r.Record(t.Context(), run(models.RunKindCron, models.RunStatusFailed, "a"))
	r.Record(
		t.Context(),
		run(models.RunKindManual, models.RunStatusSuccess, ""),
	)
	require.Len(t, al.texts, 2)
	assert.Contains(t, al.texts[1], "снова работает")
	assert.Contains(t, al.texts[1], "(manual)")
}

func TestTestAndLKIgnored(t *testing.T) {
	r, st, al := setup()
	r.Record(t.Context(), run(models.RunKindTest, models.RunStatusFailed, "a"))
	r.Record(t.Context(), run(models.RunKindLK, models.RunStatusFailed, "a"))
	assert.Empty(t, al.texts)
	assert.Zero(t, st.lastCalls)
	assert.Len(t, st.runs, 2)
}

func TestLKDoesNotMuteCronFailure(t *testing.T) {
	r, _, al := setup()
	r.Record(t.Context(), run(models.RunKindCron, models.RunStatusSuccess, ""))
	r.Record(t.Context(), run(models.RunKindLK, models.RunStatusFailed, "x"))
	r.Record(t.Context(), run(models.RunKindCron, models.RunStatusFailed, "y"))
	require.Len(t, al.texts, 1)
	assert.Contains(t, al.texts[0], "y")
}

func TestInsertErrorStillAlerts(t *testing.T) {
	r, st, al := setup()

	st.insertErr = errors.New("db down")

	assert.NotPanics(
		t,
		func() { r.Record(t.Context(), run(models.RunKindCron, models.RunStatusFailed, "a")) },
	)
	assert.Len(t, al.texts, 1)
}

func TestErrorEscapedAndTruncated(t *testing.T) {
	r, _, al := setup()
	r.Record(
		t.Context(),
		run(models.RunKindCron, models.RunStatusFailed, "<script>"),
	)
	assert.Contains(t, al.texts[0], "&lt;script&gt;")
	assert.NotContains(t, al.texts[0], "<script>")

	r2, _, al2 := setup()
	r2.Record(
		t.Context(),
		run(
			models.RunKindCron,
			models.RunStatusFailed,
			strings.Repeat("я", 600),
		),
	)
	assert.Contains(t, al2.texts[0], "<pre>"+strings.Repeat("я", 500)+"…</pre>")
}

func TestStartCleansUp(t *testing.T) {
	r, st, _ := setup()
	ctx, cancel := context.WithCancel(t.Context())

	defer cancel()

	r.Start(ctx)

	select {
	case before := <-st.deleteCall:
		assert.WithinDuration(
			t,
			time.Now().Add(-720*time.Hour),
			before,
			time.Minute,
		)

	case <-time.After(time.Second):
		t.Fatal("очистка не вызвана")
	}
}

type fakeTG struct {
	chat models.TgChat
	text string
}

func (f *fakeTG) SendText(
	_ context.Context,
	c models.TgChat,
	s string,
) (*models.SentMessage, error) {
	f.chat, f.text = c, s

	return &models.SentMessage{}, nil
}

type fakeAdmins struct{ text string }

func (f *fakeAdmins) SendAdminNotify(_ context.Context, m string) { f.text = m }

func TestTgAlerter(t *testing.T) {
	tg, ad := &fakeTG{}, &fakeAdmins{}
	require.NoError(
		t,
		NewTgAlerter(
			AlertsConfig{ChatID: 5, ThreadID: 7},
			tg,
			ad,
		).Alert(t.Context(), "x"),
	)
	assert.Equal(t, models.TgChat{ChatID: 5, ThreadID: 7}, tg.chat)
	assert.Empty(t, ad.text)

	tg, ad = &fakeTG{}, &fakeAdmins{}
	require.NoError(
		t,
		NewTgAlerter(AlertsConfig{}, tg, ad).Alert(t.Context(), "y"),
	)
	assert.Equal(t, "y", ad.text)
	assert.Empty(t, tg.text)
}

func TestRecordCountsRun(t *testing.T) {
	r, st, _ := setup()

	st.insertErr = errors.New("db down") // метрика пишется и при ошибке БД

	c := metrics.ReportRuns.WithLabelValues("cron", "failed")
	before := testutil.ToFloat64(c)

	r.Record(
		t.Context(),
		run(models.RunKindCron, models.RunStatusFailed, "boom"),
	)

	assert.InDelta(t, 1, testutil.ToFloat64(c)-before, 0)
}

func TestRecordRedactsSecrets(t *testing.T) {
	secret.Register("tok-12345")

	r, st, al := setup()
	r.Record(
		t.Context(),
		run(
			models.RunKindCron,
			models.RunStatusFailed,
			"GET /bottok-12345/x: boom",
		),
	)

	require.Len(t, st.runs, 1)
	assert.Contains(t, st.runs[0].Error, "***")
	assert.NotContains(t, st.runs[0].Error, "tok-12345")
	require.Len(t, al.texts, 1)
	assert.Contains(t, al.texts[0], "***")
	assert.NotContains(t, al.texts[0], "tok-12345")
}
