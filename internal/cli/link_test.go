package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"support_bot/internal/models"
)

type fakeLinks struct {
	ttl     time.Duration
	report  string
	links   []models.PublicLink
	revoked bool
	err     error
}

func (f *fakeLinks) CreatePublicLink(
	_ context.Context,
	r string,
	ttl time.Duration,
) (string, error) {
	f.report, f.ttl = r, ttl

	return "uuid-1", f.err
}

func (f *fakeLinks) ListPublicLinks(
	context.Context,
	string,
) ([]models.PublicLink, error) {
	return f.links, f.err
}

func (f *fakeLinks) ExpirePublicLink(context.Context, string) (bool, error) {
	return f.revoked, f.err
}

func openFake(f *fakeLinks) openLinkStore {
	return func(context.Context, string) (linkStore, func(), error) { return f, func() {}, nil }
}

func TestLinkCreate(t *testing.T) {
	f := &fakeLinks{}

	var out bytes.Buffer

	if err := linkCreate(
		t.Context(),
		openFake(f),
		&out,
		[]string{"-report", "r", "-ttl", "168h"},
	); err != nil {
		t.Fatal(err)
	}

	if f.report != "r" || f.ttl != 168*time.Hour {
		t.Fatalf("got %q %v", f.report, f.ttl)
	}

	if !strings.Contains(out.String(), "/api/v1/public/report/uuid-1") {
		t.Fatalf("output: %q", out.String())
	}

	if err := linkCreate(t.Context(), openFake(f), &out, nil); err == nil {
		t.Fatal("want error without -report")
	}

	f.err = models.ErrNotFound
	if err := linkCreate(
		t.Context(),
		openFake(f),
		&out,
		[]string{"-report", "x"},
	); err == nil {
		t.Fatal("want error for missing report")
	}
}

func TestLinkList(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	f := &fakeLinks{links: []models.PublicLink{
		{ID: "forever", Report: "r", CreatedAt: time.Now()},
		{ID: "old", Report: "r", CreatedAt: time.Now(), ExpiresAt: &past},
	}}

	var out bytes.Buffer

	if err := linkList(t.Context(), openFake(f), &out, nil); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("output: %q", out.String())
	}

	if !strings.Contains(lines[1], " - ") ||
		!strings.Contains(lines[1], "активна") {
		t.Errorf("forever: %q", lines[1])
	}

	if !strings.Contains(lines[2], "истекла") {
		t.Errorf("old: %q", lines[2])
	}
}

func TestLinkRevoke(t *testing.T) {
	var out bytes.Buffer

	f := &fakeLinks{revoked: true}

	if err := linkRevoke(
		t.Context(),
		openFake(f),
		&out,
		[]string{"id"},
	); err != nil ||
		!strings.Contains(out.String(), "отозвана") {
		t.Fatalf("revoked: %v %q", err, out.String())
	}

	out.Reset()

	f.revoked = false
	if err := linkRevoke(
		t.Context(),
		openFake(f),
		&out,
		[]string{"id"},
	); err != nil ||
		!strings.Contains(out.String(), "уже неактивна") {
		t.Fatalf("inactive: %v %q", err, out.String())
	}

	f.err = models.ErrNotFound
	if err := linkRevoke(
		t.Context(),
		openFake(f),
		&out,
		[]string{"id"},
	); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Fatalf("not found: %v", err)
	}

	if err := linkRevoke(t.Context(), openFake(f), &out, nil); err == nil {
		t.Fatal("want error without uuid")
	}
}
