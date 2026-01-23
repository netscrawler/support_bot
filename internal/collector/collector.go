package collector

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	models "support_bot/internal/models/report"
)

const defaultParralellCollectors = 32

type DataFetcher interface {
	Fetch(ctx context.Context, uuid string) ([]map[string]any, error)
}

type Collector struct {
	mb       DataFetcher
	log      *slog.Logger
	parallel chan struct{}
}

func NewCollector(parralell uint8, mb DataFetcher, log *slog.Logger) *Collector {
	l := log.With(slog.Any("module", "collector"))

	if parralell == 0 {
		parralell = defaultParralellCollectors
	}

	l.Info("create collector", slog.Any("parralell_query", parralell))
	semaphor := make(chan struct{}, parralell)

	return &Collector{
		mb:       mb,
		log:      l,
		parallel: semaphor,
	}
}

func (c *Collector) Collect(
	ctx context.Context,
	cards ...models.Card,
) (map[string][]map[string]any, error) {
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

		return nil, errors.New("empty cards")
	}

	collected := make(map[string][]map[string]any)

	var wg sync.WaitGroup

	resChan, errChan := c.collect(ctx, &wg, cards...)

	wg.Wait()
	close(resChan)
	close(errChan)

	for r := range resChan {
		c.log.DebugContext(ctx, "collecting data from card", slog.Any("card_name", r.Name))
		collected[r.Name] = r.Data
	}

	var collectError error

	for err := range errChan {
		c.log.DebugContext(ctx, "error while collecting data from card", slog.Any("error", err))
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

			data, err := c.mb.Fetch(ctx, crd.CardUUID)
			if err != nil {
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
