package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"support_bot/internal/metrics"
	"support_bot/internal/models"
)

const (
	defaultParallelCollectors = 32
)

type DataFetcher interface {
	Fetch(
		ctx context.Context,
		target string,
		params map[string]string,
	) ([]map[string]any, error)
}

// QueryFetcher выполняет запрос query в именованном источнике target.
type QueryFetcher interface {
	Fetch(
		ctx context.Context,
		target, query string,
		params map[string]string,
	) ([]map[string]any, error)
}

// Fetchers — реестр источников по типу карточки.
type Fetchers map[string]QueryFetcher

type ignoreQuery struct{ f DataFetcher }

func (q ignoreQuery) Fetch(
	ctx context.Context,
	target, _ string,
	params map[string]string,
) ([]map[string]any, error) {
	return q.f.Fetch(ctx, target, params)
}

// IgnoreQuery приводит источник без query к QueryFetcher.
func IgnoreQuery(f DataFetcher) QueryFetcher { return ignoreQuery{f} }

type Collector struct {
	fetchers Fetchers
	log      *slog.Logger
	parallel chan struct{}
}

func NewCollector(
	parallel uint8,
	fetchers Fetchers,
	log *slog.Logger,
) *Collector {
	l := log.With(slog.Any("module", "collector"))

	if parallel == 0 {
		parallel = defaultParallelCollectors
	}

	l.Info("create collector", slog.Any("parallel_query", parallel))

	return &Collector{
		fetchers: fetchers,
		log:      l,
		parallel: make(chan struct{}, parallel),
	}
}

func (c *Collector) Collect(
	ctx context.Context,
	cards ...models.Card,
) (models.Dataset, error) {
	start := time.Now()

	c.log.InfoContext(ctx, "Start collecting data")

	defer func() {
		c.log.InfoContext(
			ctx,
			"Finish collecting data",
			slog.Any("time elapsed", time.Since(start)),
		)
	}()

	if len(cards) == 0 {
		c.log.ErrorContext(ctx, "empty card list")

		return nil, ErrEmtyCard
	}

	collected := make(models.Dataset)

	var wg sync.WaitGroup

	resChan, errChan := c.collect(ctx, &wg, cards...)

	wg.Wait()
	close(resChan)
	close(errChan)

	for r := range resChan {
		c.log.DebugContext(
			ctx,
			"collecting data from card",
			slog.Any("card_name", r.Name),
		)

		collected[r.Name] = r.Data
	}

	var collectError error

	for err := range errChan {
		c.log.DebugContext(
			ctx,
			"error while collecting data from card",
			slog.Any("error", err),
		)

		collectError = errors.Join(collectError, err)
	}

	return collected, collectError
}

type res struct {
	Name string
	Data []map[string]any
}

func (c *Collector) collect(
	ctx context.Context,
	wg *sync.WaitGroup,
	cards ...models.Card,
) (chan res, chan error) {
	resChan := make(chan res, len(cards))
	errChan := make(chan error, len(cards))

	for _, crd := range cards {
		wg.Add(1)

		c.parallel <- struct{}{}

		go func(crd models.Card) {
			defer wg.Done()
			defer func() { <-c.parallel }()

			f := c.fetchers[crd.Type]

			fetchStart := time.Now()

			var (
				data []map[string]any
				err  error
			)

			if f == nil {
				err = fmt.Errorf("unknown fetcher type %s", crd.Type)
			} else {
				data, err = f.Fetch(ctx, crd.CardUUID, crd.Query, crd.Params)
			}

			rep := metrics.ReportFrom(ctx)

			metrics.CollectorDuration.WithLabelValues(crd.Type, rep, crd.Title).
				Observe(time.Since(fetchStart).Seconds())

			if err != nil {
				metrics.CollectorErrors.WithLabelValues(crd.Type, rep, crd.Title, metrics.Reason(err)).
					Inc()
				c.log.ErrorContext(
					ctx,
					"Error while fetching data",
					slog.Any("card", crd),
					slog.Any("error", err),
				)

				errChan <- err
			}

			resChan <- res{Name: crd.Title, Data: data}
		}(crd)
	}

	return resChan, errChan
}
