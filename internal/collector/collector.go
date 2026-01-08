package collector

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

const defaultParralellCollectors = 32

type DataFetcher interface {
	Fetch(ctx context.Context, uuid string) ([]map[string]any, error)
	FetchMatrix(ctx context.Context, uuid string) ([][]string, error)
}

type Card struct {
	Name string
	UUID string
}

type Collector struct {
	mb       DataFetcher
	log      *slog.Logger
	parallel chan struct{}
}

func NewCollector(parralell uint8, mb DataFetcher, log *slog.Logger) *Collector {
	if parralell == 0 {
		parralell = defaultParralellCollectors
	}
	semaphor := make(chan struct{}, parralell)
	return &Collector{
		mb:       mb,
		log:      log,
		parallel: semaphor,
	}
}

func (c *Collector) Collect(
	ctx context.Context,
	cards ...Card,
) (map[string][]map[string]any, error) {
	c.log.DebugContext(ctx, "Start collecting data")

	if len(cards) == 0 {
		return nil, errors.New("empty cards")
	}

	collected := make(map[string][]map[string]any)

	var wg sync.WaitGroup

	resChan, errChan := c.collect(ctx, &wg, cards...)

	wg.Wait()
	close(resChan)
	close(errChan)

	for r := range resChan {
		collected[r.Name] = r.Data
	}

	var collectError error
	for err := range errChan {
		collectError = errors.Join(collectError, err)
	}

	return collected, collectError
}

func (c *Collector) CollectTables(
	ctx context.Context,
	cards ...Card,
) (map[string][][]string, error) {
	c.log.DebugContext(ctx, "Start collecting data")

	if len(cards) == 0 {
		return nil, errors.New("empty cards")
	}

	collected := make(map[string][][]string)

	var wg sync.WaitGroup

	resChan, errChan := c.collectTables(ctx, &wg, cards...)

	wg.Wait()
	close(resChan)
	close(errChan)

	for r := range resChan {
		collected[r.Name] = r.Data
	}

	var collectError error
	for err := range errChan {
		collectError = errors.Join(collectError, err)
	}

	return collected, collectError
}

type res struct {
	Name string
	Data []map[string]any
}
type resTable struct {
	Name string
	Data [][]string
}

func (c *Collector) collectTables(
	ctx context.Context,
	wg *sync.WaitGroup,
	cards ...Card,
) (chan resTable, chan error) {
	resChan := make(chan resTable, len(cards))
	errChan := make(chan error, len(cards))

	for _, crd := range cards {
		wg.Add(1)
		c.parallel <- struct{}{}

		go func(crd Card) {
			defer wg.Done()
			defer func() { <-c.parallel }()
			data, err := c.mb.FetchMatrix(ctx, crd.UUID)
			if err != nil {
				c.log.ErrorContext(
					ctx,
					"Error while fetching data",
					slog.Any("card", crd),
					slog.Any("error", err),
				)
				errChan <- err
			}
			resChan <- resTable{Name: crd.Name, Data: data}
		}(crd)
	}

	return resChan, errChan
}

func (c *Collector) collect(
	ctx context.Context,
	wg *sync.WaitGroup,
	cards ...Card,
) (chan res, chan error) {
	resChan := make(chan res, len(cards))
	errChan := make(chan error, len(cards))

	for _, crd := range cards {
		wg.Add(1)
		c.parallel <- struct{}{}

		go func(crd Card) {
			defer wg.Done()
			defer func() { <-c.parallel }()
			data, err := c.mb.Fetch(ctx, crd.UUID)
			if err != nil {
				c.log.ErrorContext(
					ctx,
					"Error while fetching data",
					slog.Any("card", crd),
					slog.Any("error", err),
				)
				errChan <- err
			}
			resChan <- res{Name: crd.Name, Data: data}
		}(crd)
	}

	return resChan, errChan
}
