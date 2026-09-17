package service

import (
	"context"
	"log/slog"
	"support_bot/internal/models"
	"testing"
)

type fakeReportDB struct {
	report *models.Report
	err    error
}

func (f fakeReportDB) GetByPublicID(_ context.Context, _ string) (*models.Report, error) {
	return f.report, f.err
}

type fakeReportGenerator struct {
	data models.Data
	err  error
}

func (f fakeReportGenerator) Generate(_ context.Context, _ models.Report) (models.Data, error) {
	return f.data, f.err
}

// TestGenerateReport_NotFoundFromDB проверяет, что GenerateReport
// пробрасывает models.ErrNotFound, если отчет с данным public_id не найден в БД.
func TestGenerateReport_NotFoundFromDB(t *testing.T) {
	r := NewReport(
		fakeReportDB{err: models.ErrNotFound},
		fakeReportGenerator{},
		slog.New(slog.DiscardHandler),
	)

	_, err := r.GenerateReport(context.Background(), "some-id")
	if err != models.ErrNotFound {
		t.Fatalf("GenerateReport() error = %v, want models.ErrNotFound", err)
	}
}

// TestGenerateReport_NotFoundFromGenerator проверяет, что GenerateReport
// пробрасывает models.ErrNotFound, если сам генератор вернул эту ошибку
// (отчет найден в БД, но генерация сообщает об отсутствии данных).
func TestGenerateReport_NotFoundFromGenerator(t *testing.T) {
	r := NewReport(
		fakeReportDB{report: &models.Report{Name: "r1"}},
		fakeReportGenerator{err: models.ErrNotFound},
		slog.New(slog.DiscardHandler),
	)

	_, err := r.GenerateReport(context.Background(), "some-id")
	if err != models.ErrNotFound {
		t.Fatalf("GenerateReport() error = %v, want models.ErrNotFound", err)
	}
}

// TestGenerateReport_ReturnsGeneratedData проверяет, что при успешном поиске
// отчета и успешной генерации GenerateReport возвращает данные без ошибки.
func TestGenerateReport_ReturnsGeneratedData(t *testing.T) {
	want := models.Data{FileName: "out.csv"}
	r := NewReport(
		fakeReportDB{report: &models.Report{Name: "r1"}},
		fakeReportGenerator{data: want},
		slog.New(slog.DiscardHandler),
	)

	got, err := r.GenerateReport(context.Background(), "some-id")
	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	if got.FileName != want.FileName {
		t.Fatalf("GenerateReport() = %+v, want %+v", got, want)
	}
}
