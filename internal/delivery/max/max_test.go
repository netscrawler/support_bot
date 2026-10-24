package maxadp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"support_bot/internal/models"
	"support_bot/internal/pkg/retry"

	maxcli "github.com/max-messenger/max-bot-api-client-go/v2"
	maxModel "github.com/max-messenger/max-bot-api-client-go/v2/model"
)

// fakeMax эмулирует API MAX: выдачу upload URL, приём файла и отправку сообщения.
type fakeMax struct {
	mu          sync.Mutex
	failUpload  map[string]bool // имя файла -> всегда 500
	failSend    bool
	sendCalls   int
	uploadCalls map[string]int
	sent        []maxModel.NewMessageBody
}

// apiError отвечает ошибкой API в JSON, иначе клиент MAX сам повторяет запрос с паузами в секунды.
func apiError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)

	_, _ = io.WriteString(w, `{"code":"internal.error","message":"boom"}`)
}

func newAdaptor(t *testing.T, f *fakeMax) *Adaptor {
	t.Helper()

	f.uploadCalls = map[string]int{}

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc(
		"POST /uploads",
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(maxModel.UploadEndpoint{
				Url: srv.URL + "/up?type=" + r.URL.Query().Get("type"),
			})
		},
	)
	mux.HandleFunc("POST /up", func(w http.ResponseWriter, r *http.Request) {
		_, hdr, err := r.FormFile("data")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		f.mu.Lock()

		f.uploadCalls[hdr.Filename]++

		fail := f.failUpload[hdr.Filename]
		f.mu.Unlock()

		if fail {
			apiError(w)

			return
		}

		tok := "tok-" + hdr.Filename
		if r.URL.Query().Get("type") == string(maxModel.UploadImage) {
			_, _ = io.WriteString(w, `{"photos":{"x":{"token":"`+tok+`"}}}`)

			return
		}

		_, _ = io.WriteString(w, `{"token":"`+tok+`"}`)
	})
	mux.HandleFunc(
		"POST /messages",
		func(w http.ResponseWriter, r *http.Request) {
			var body maxModel.NewMessageBody

			_ = json.NewDecoder(r.Body).Decode(&body)

			f.mu.Lock()

			f.sendCalls++

			f.sent = append(f.sent, body)

			fail := f.failSend
			f.mu.Unlock()

			if fail {
				apiError(w)

				return
			}

			_, _ = io.WriteString(w, `{"message":{"body":{"mid":"m1"}}}`)
		},
	)

	api, err := maxcli.NewApi("token", maxcli.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}

	return New(
		api,
		retry.Delays{time.Millisecond, time.Millisecond},
		true,
		slog.New(slog.DiscardHandler),
	)
}

func file(name string) models.Data {
	return models.Data{Data: bytes.NewBufferString("data"), FileName: name}
}

func TestSendUploadsPartial(t *testing.T) {
	f := &fakeMax{failUpload: map[string]bool{"bad.txt": true}}
	adp := newAdaptor(t, f)

	msg, err := adp.SendDocument(
		context.Background(),
		models.MaxChat{ChatID: 1},
		[]models.Data{file("bad.txt"), file("good.txt")},
	)
	if err == nil || !strings.Contains(err.Error(), "bad.txt") {
		t.Fatalf("want error mentioning bad.txt, got %v", err)
	}

	if msg == nil {
		t.Fatal("want message with the uploaded part")
	}

	if f.sendCalls != 1 || len(f.sent[0].Attachments) != 1 {
		t.Fatalf("sends=%d sent=%+v", f.sendCalls, f.sent)
	}

	if f.uploadCalls["bad.txt"] != 3 {
		t.Fatalf("want 3 upload attempts, got %d", f.uploadCalls["bad.txt"])
	}
}

func TestSendUploadsAllFail(t *testing.T) {
	f := &fakeMax{failUpload: map[string]bool{"a.txt": true, "b.txt": true}}
	adp := newAdaptor(t, f)

	msg, err := adp.SendDocument(
		context.Background(),
		models.MaxChat{ChatID: 1},
		[]models.Data{file("a.txt"), file("b.txt")},
	)
	if err == nil {
		t.Fatal("want error")
	}

	if msg != nil || f.sendCalls != 0 {
		t.Fatalf("msg=%v sends=%d", msg, f.sendCalls)
	}
}

func TestSendUploadsAttachType(t *testing.T) {
	f := &fakeMax{}
	adp := newAdaptor(t, f)
	chat := models.MaxChat{ChatID: 1}

	if _, err := adp.SendDocument(
		context.Background(),
		chat,
		[]models.Data{file("d.txt")},
	); err != nil {
		t.Fatal(err)
	}

	if _, err := adp.SendMedia(
		context.Background(),
		chat,
		[]models.Data{file("i.png")},
	); err != nil {
		t.Fatal(err)
	}

	if got := f.sent[0].Attachments[0].Type; got != maxModel.AttachFile {
		t.Fatalf("document attached as %q", got)
	}

	if got := f.sent[1].Attachments[0].Type; got != maxModel.AttachImage {
		t.Fatalf("media attached as %q", got)
	}
}

func TestSendTextRetriesAndReturnsLastError(t *testing.T) {
	f := &fakeMax{failSend: true}
	adp := newAdaptor(t, f)

	_, err := adp.SendText(
		context.Background(),
		models.MaxChat{ChatID: 1},
		"hi",
	)
	if err == nil {
		t.Fatal("want error")
	}

	if f.sendCalls != 3 {
		t.Fatalf("want 3 attempts, got %d", f.sendCalls)
	}
}

func TestSendTextCancelledCtxStopsRetries(t *testing.T) {
	f := &fakeMax{failSend: true}
	adp := newAdaptor(t, f)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := adp.SendText(
		ctx,
		models.MaxChat{ChatID: 1},
		"hi",
	); err == nil {
		t.Fatal("want error")
	}

	if f.sendCalls > 1 {
		t.Fatalf("retried after cancel: %d calls", f.sendCalls)
	}
}
