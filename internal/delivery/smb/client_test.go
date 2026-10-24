package smb

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"support_bot/internal/models"
)

type fakeShare struct {
	failures int // сколько первых записей вернут ошибку
	writes   int
	written  []byte
	umounted bool
}

func (f *fakeShare) WriteFile(_ string, data []byte, _ os.FileMode) error {
	f.writes++
	if f.writes <= f.failures {
		return errors.New("write failed")
	}

	f.written = data

	return nil
}

func (f *fakeShare) Stat(string) (os.FileInfo, error) {
	return nil, os.ErrNotExist
}

func (f *fakeShare) Umount() error {
	f.umounted = true

	return nil
}

func newTestSMB(fs share) *SMB {
	return &SMB{
		cfg:    Config{Active: true},
		delays: []time.Duration{time.Millisecond, time.Millisecond},
		fs:     fs,
		log:    slog.New(slog.DiscardHandler),
	}
}

func testData() models.Data {
	return models.Data{FileName: "a.txt", Data: bytes.NewBufferString("hello")}
}

func TestCleanupKeepsMonitorContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs := &fakeShare{}
	s := newTestSMB(fs)

	s.cancel = cancel

	s.cleanup()

	if ctx.Err() != nil {
		t.Fatal("cleanup отменил контекст монитора")
	}

	if !fs.umounted || s.fs != nil {
		t.Fatal("cleanup не закрыл fs")
	}
}

func TestCloseCancelsMonitorContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := newTestSMB(&fakeShare{})

	s.cancel = cancel

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	if ctx.Err() == nil {
		t.Fatal("Close не отменил контекст монитора")
	}
}

func TestUploadRetriesAfterError(t *testing.T) {
	fs := &fakeShare{failures: 2}
	s := newTestSMB(fs)

	if err := s.Upload(context.Background(), "dir", testData()); err != nil {
		t.Fatal(err)
	}

	if fs.writes != 3 || string(fs.written) != "hello" {
		t.Fatalf("writes=%d written=%q", fs.writes, fs.written)
	}
}

func TestUploadCancelledContextStopsRetries(t *testing.T) {
	fs := &fakeShare{failures: 100}
	s := newTestSMB(fs)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.Upload(ctx, "dir", testData()); err == nil {
		t.Fatal("ожидалась ошибка")
	}

	if fs.writes != 1 {
		t.Fatalf("после отмены ctx записей: %d, ожидалась 1", fs.writes)
	}
}

func TestUploadNilShare(t *testing.T) {
	s := newTestSMB(nil)

	if err := s.Upload(context.Background(), "dir", testData()); err == nil {
		t.Fatal("ожидалась ошибка smb not connected")
	}
}

func TestUploadConcurrentWithCleanup(*testing.T) {
	s := newTestSMB(&fakeShare{})

	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = s.Upload(context.Background(), "dir", testData())
	}()

	s.cleanup()
	<-done
}
