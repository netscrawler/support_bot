package store

import (
	"context"
	"reflect"
	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestCronStore_LoadActive_MapsCronNameAndEventType(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("select cron, name, event_type from crons where is_active = true").
		WillReturnRows(pgxmock.NewRows([]string{"cron", "name", "event_type"}).
			AddRow("0 9 * * 1", "weekly", int32(models.EventTypeGenReport)))

	store := &CronStore{q: sqlcgen.New(pool)}
	got, err := store.LoadActive(context.Background())
	if err != nil {
		t.Fatalf("LoadActive() error = %v", err)
	}
	want := []models.SheduleUnit{{
		Crontab: "0 9 * * 1", Name: "weekly", EventType: int(models.EventTypeGenReport),
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadActive() = %#v, want %#v", got, want)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCronStore_LoadEvents_MapsActiveReportLinks(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("from report_crons rc join crons c on c.id = rc.cron_id join reports r on r.id = rc.report_id where r.active = true").
		WillReturnRows(pgxmock.NewRows([]string{"cron_name", "report_name"}).AddRow("weekly", "sales"))

	store := &CronStore{q: sqlcgen.New(pool)}
	got, err := store.LoadEvents(context.Background())
	if err != nil {
		t.Fatalf("LoadEvents() error = %v", err)
	}
	want := []models.ReportCronEvent{{CronName: "weekly", Name: "sales"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadEvents() = %#v, want %#v", got, want)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCronStore_LoadEventsByCronName_MapsOnlyRequestedCron(t *testing.T) {
	pool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer pool.Close()

	pool.ExpectQuery("where c.name = \\$1 and r.active = true").WithArgs("weekly").
		WillReturnRows(pgxmock.NewRows([]string{"cron_name", "report_name"}).AddRow("weekly", "sales"))

	store := &CronStore{q: sqlcgen.New(pool)}
	got, err := store.LoadEventsByCronName(context.Background(), "weekly")
	if err != nil {
		t.Fatalf("LoadEventsByCronName() error = %v", err)
	}
	want := []models.ReportCronEvent{{CronName: "weekly", Name: "sales"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadEventsByCronName() = %#v, want %#v", got, want)
	}
	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
