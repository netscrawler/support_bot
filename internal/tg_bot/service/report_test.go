package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	eventcreator "support_bot/internal/event_creator"
	"support_bot/internal/models"
	service_mock "support_bot/internal/tg_bot/service/mock"
)

func TestGenerateReportPeriod(t *testing.T) {
	outSp := make(chan models.SpecialEventForLK, 2)
	svc := &Report{
		EventAPI: eventcreator.NewEventAPI(nil, outSp),
		log:      slog.Default(),
	}
	chat := &models.Chat{ChatID: 1}

	ctx := context.Background()
	y := models.PeriodYesterday

	if err := svc.GenerateReportByName(ctx, "r", chat, y); err != nil {
		t.Fatal(err)
	}

	if err := svc.GenerateTestReportByName(ctx, "r", chat, y); err != nil {
		t.Fatal(err)
	}

	seen := map[bool]bool{}

	for range 2 {
		select {
		case ev := <-outSp:
			seen[ev.Test] = true

			if ev.Period != y {
				t.Errorf("period=%q, want yesterday", ev.Period)
			}

		case <-time.After(time.Second):
			t.Fatal("no event")
		}
	}

	if !seen[true] || !seen[false] {
		t.Errorf("want both test and regular events, got %v", seen)
	}
}

func TestUsesPeriod(t *testing.T) {
	repo := service_mock.NewMockReportProvider(t)
	repo.EXPECT().GetByName(context.Background(), "r").Return(&models.Report{
		Queries: []models.Card{
			{RawParams: []byte(`{"from":"{{period_from}}"}`)},
		},
	}, nil)

	svc := &Report{repo: repo}

	got, err := svc.UsesPeriod(context.Background(), "r")
	if err != nil || !got {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
