package metrics

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// RegisterPool регистрирует метрики пула соединений pgx; повторный вызов безопасен.
func RegisterPool(stat func() *pgxpool.Stat) {
	gauge := func(name, help string, f func(*pgxpool.Stat) float64) prometheus.Collector {
		return prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{Name: name, Help: help},
			func() float64 { return f(stat()) },
		)
	}
	counter := func(name, help string, f func(*pgxpool.Stat) float64) prometheus.Collector {
		return prometheus.NewCounterFunc(
			prometheus.CounterOpts{Name: name, Help: help},
			func() float64 { return f(stat()) },
		)
	}

	for _, c := range []prometheus.Collector{
		gauge("sbot_db_pool_acquired_conns", "Число занятых соединений пула БД.",
			func(s *pgxpool.Stat) float64 { return float64(s.AcquiredConns()) }),
		gauge("sbot_db_pool_idle_conns", "Число простаивающих соединений пула БД.",
			func(s *pgxpool.Stat) float64 { return float64(s.IdleConns()) }),
		gauge("sbot_db_pool_total_conns", "Общее число соединений пула БД.",
			func(s *pgxpool.Stat) float64 { return float64(s.TotalConns()) }),
		gauge("sbot_db_pool_max_conns", "Максимальный размер пула БД.",
			func(s *pgxpool.Stat) float64 { return float64(s.MaxConns()) }),
		counter("sbot_db_pool_acquire_total", "Число успешных получений соединения из пула БД.",
			func(s *pgxpool.Stat) float64 { return float64(s.AcquireCount()) }),
		counter("sbot_db_pool_empty_acquire_total", "Число получений соединения при пустом пуле БД (с ожиданием).",
			func(s *pgxpool.Stat) float64 { return float64(s.EmptyAcquireCount()) }),
		counter("sbot_db_pool_acquire_wait_seconds_total", "Суммарное время получения соединения из пула БД, секунды.",
			func(s *pgxpool.Stat) float64 { return s.AcquireDuration().Seconds() }),
	} {
		if err := prometheus.Register(c); err != nil {
			if _, ok := errors.AsType[prometheus.AlreadyRegisteredError](
				err,
			); !ok {
				panic(err)
			}
		}
	}
}
