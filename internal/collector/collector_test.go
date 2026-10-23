package collector_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"support_bot/internal/collector"
	"support_bot/internal/metrics"
	"support_bot/internal/models"
	"sync"
	"testing"
	"time"

	cmock "support_bot/internal/collector/mock"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	th := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(th)

	t.Run("one query", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		df := &cmock.MockDataFetcher{}

		card := models.Card{Title: "card1", CardUUID: "uuid1", Type: models.CollectTypeMetabase}

		df.On("Fetch", ctx, "uuid1", mock.Anything).Return([]map[string]any{
			{"field": "value1"},
		}, nil)

		c := collector.NewCollector(0, df, df, df, nil, nil, nil, nil, l)

		result, err := c.Collect(ctx, card)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "value1", result["card1"][0]["field"])

		df.AssertExpectations(t)
	})

	t.Run("pg card", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		pg := &cmock.MockQueryFetcher{}
		params := map[string]string{"a": "b"}
		card := models.Card{
			Title: "card1", CardUUID: "analytics", Type: models.CollectTypePostgres,
			Query: "select 1", Params: params,
		}

		pg.On("Fetch", ctx, "analytics", "select 1", params).
			Return([]map[string]any{{"field": "pg"}}, nil)

		c := collector.NewCollector(0, nil, nil, nil, pg, nil, nil, nil, l)

		result, err := c.Collect(ctx, card)

		require.NoError(t, err)
		assert.Equal(t, "pg", result["card1"][0]["field"])
		pg.AssertExpectations(t)
	})

	t.Run("prom card", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		prom := &cmock.MockQueryFetcher{}
		params := map[string]string{"a": "b"}
		card := models.Card{
			Title: "card1", CardUUID: "main", Type: models.CollectTypePrometheus,
			Query: "up", Params: params,
		}

		prom.On("Fetch", ctx, "main", "up", params).
			Return([]map[string]any{{"field": "prom"}}, nil)

		c := collector.NewCollector(0, nil, nil, nil, nil, prom, nil, nil, l)

		result, err := c.Collect(ctx, card)

		require.NoError(t, err)
		assert.Equal(t, "prom", result["card1"][0]["field"])
		prom.AssertExpectations(t)
	})

	t.Run("grafana alerts card", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		alerts := &cmock.MockQueryFetcher{}
		params := map[string]string{"a": "b"}
		card := models.Card{
			Title: "card1", CardUUID: "ops", Type: models.CollectTypeGrafanaAlerts,
			Query: "active", Params: params,
		}

		alerts.On("Fetch", ctx, "ops", "active", params).
			Return([]map[string]any{{"field": "alerts"}}, nil)

		c := collector.NewCollector(0, nil, nil, nil, nil, nil, alerts, nil, l)

		result, err := c.Collect(ctx, card)

		require.NoError(t, err)
		assert.Equal(t, "alerts", result["card1"][0]["field"])
		alerts.AssertExpectations(t)
	})

	t.Run("zabbix card", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		zbx := &cmock.MockQueryFetcher{}
		params := map[string]string{"a": "b"}
		card := models.Card{
			Title: "card1", CardUUID: "host1", Type: models.CollectTypeZabbix,
			Query: "item.key", Params: params,
		}

		zbx.On("Fetch", ctx, "host1", "item.key", params).
			Return([]map[string]any{{"field": "zabbix"}}, nil)

		c := collector.NewCollector(0, nil, nil, nil, nil, nil, nil, zbx, l)

		result, err := c.Collect(ctx, card)

		require.NoError(t, err)
		assert.Equal(t, "zabbix", result["card1"][0]["field"])
		zbx.AssertExpectations(t)
	})

	t.Run("many query", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		df := &cmock.MockDataFetcher{}

		cards := []models.Card{
			{Title: "card1", CardUUID: "uuid1", Type: models.CollectTypeMetabase},
			{Title: "card2", CardUUID: "uuid2", Type: models.CollectTypeMetabase},
			{Title: "card3", CardUUID: "uuid3", Type: models.CollectTypeMetabase},
		}

		for _, card := range cards {
			uuid := card.CardUUID
			df.On("Fetch", ctx, uuid, mock.Anything).Run(func(_ mock.Arguments) {
				time.Sleep(1 * time.Second)
			}).Return([]map[string]any{{"field": "value_" + uuid}}, nil)
		}

		c := collector.NewCollector(3, df, df, df, nil, nil, nil, nil, l)

		start := time.Now()
		result, err := c.Collect(ctx, cards...)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, result, 3)

		for _, card := range cards {
			assert.Equal(t, "value_"+card.CardUUID, result[card.Title][0]["field"])
		}

		df.AssertExpectations(t)

		assert.Less(t, duration.Seconds(), 1.5, "Tasks should run in parallel")
	})

	t.Run("many query with one err", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		df := &cmock.MockDataFetcher{}

		cards := []models.Card{
			{Title: "card1", CardUUID: "uuid1", Type: models.CollectTypeMetabase},
			{Title: "card2", CardUUID: "uuid2", Type: models.CollectTypeMetabase},
		}

		df.On("Fetch", ctx, "uuid1", mock.Anything).Run(func(_ mock.Arguments) {
			time.Sleep(1 * time.Second)
		}).Return([]map[string]any{{"field": "value_" + "uuid1"}}, nil)

		df.On("Fetch", ctx, "uuid2", mock.Anything).Run(func(_ mock.Arguments) {
			time.Sleep(1 * time.Second)
		}).Return(nil, errors.New("some error"))

		c := collector.NewCollector(3, df, df, df, nil, nil, nil, nil, l)

		start := time.Now()
		result, err := c.Collect(ctx, cards...)
		duration := time.Since(start)

		require.EqualError(t, err, "some error")
		assert.Len(t, result, 2)

		assert.Equal(t, "value_uuid1", result["card1"][0]["field"])
		assert.Empty(t, result["card2"])

		df.AssertExpectations(t)

		assert.Less(t, duration.Seconds(), 1.5, "Tasks should run in parallel")
	})
	t.Run("single collector multiple goroutines", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		df := &cmock.MockDataFetcher{}

		cards1 := []models.Card{
			{Title: "card1", CardUUID: "uuid1", Type: models.CollectTypeMetabase},
			{Title: "card2", CardUUID: "uuid2", Type: models.CollectTypeMetabase},
		}
		cards2 := []models.Card{
			{Title: "card3", CardUUID: "uuid3", Type: models.CollectTypeMetabase},
			{Title: "card4", CardUUID: "uuid4", Type: models.CollectTypeMetabase},
		}

		for _, card := range append(cards1, cards2...) {
			df.On("Fetch", ctx, card.CardUUID, mock.Anything).Run(func(_ mock.Arguments) {
				time.Sleep(200 * time.Millisecond) // симуляция долгой работы
			}).Return([]map[string]any{{"field": "value_" + card.CardUUID}}, nil)
		}

		c := collector.NewCollector(2, df, df, df, nil, nil, nil, nil, slog.Default())

		start := time.Now()

		var wg sync.WaitGroup

		var (
			result1, result2 map[string][]map[string]any
			err1, err2       error
		)

		wg.Go(func() {
			result1, err1 = c.Collect(ctx, cards1...)
		})

		wg.Go(func() {
			result2, err2 = c.Collect(ctx, cards2...)
		})

		wg.Wait()

		duration := time.Since(start)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Len(t, result1, 2)
		assert.Len(t, result2, 2)

		for _, card := range cards1 {
			expected := "value_" + card.CardUUID
			assert.Equal(
				t,
				expected,
				result1[card.Title][0]["field"],
				"card in result1",
			)
		}

		for _, card := range cards2 {
			expected := "value_" + card.CardUUID
			assert.Equal(
				t,
				expected,
				result2[card.Title][0]["field"],
				"card in result2",
			)
		}

		assert.Less(t, duration.Seconds(), 0.6, "Semaphore should limit concurrent execution")

		df.AssertExpectations(t)
	})
	t.Run("semaphore enforces parallelism across goroutines", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		df := &cmock.MockDataFetcher{}

		cards := []models.Card{
			{Title: "card1", CardUUID: "uuid1", Type: models.CollectTypeMetabase},
			{Title: "card2", CardUUID: "uuid2", Type: models.CollectTypeMetabase},
			{Title: "card3", CardUUID: "uuid3", Type: models.CollectTypeMetabase},
			{Title: "card4", CardUUID: "uuid4", Type: models.CollectTypeMetabase},
			{Title: "card5", CardUUID: "uuid5", Type: models.CollectTypeMetabase},
			{Title: "card6", CardUUID: "uuid6", Type: models.CollectTypeMetabase},
		}

		var mu sync.Mutex

		currentRunning := 0
		maxRunning := 0

		for _, card := range cards {
			uuid := card.CardUUID
			df.On("Fetch", ctx, uuid, mock.Anything).Run(func(_ mock.Arguments) {
				mu.Lock()

				currentRunning++
				if currentRunning > maxRunning {
					maxRunning = currentRunning
				}

				mu.Unlock()

				time.Sleep(100 * time.Millisecond)

				mu.Lock()

				currentRunning--

				mu.Unlock()
			}).Return([]map[string]any{{"field": "value_" + uuid}}, nil)
		}

		c := collector.NewCollector(3, df, df, df, nil, nil, nil, nil, slog.Default())

		var wg sync.WaitGroup

		var (
			result1, result2 map[string][]map[string]any
			err1, err2       error
		)

		wg.Go(func() {
			result1, err1 = c.Collect(ctx, cards[:3]...)
		})
		wg.Go(func() {
			result2, err2 = c.Collect(ctx, cards[3:]...)
		})

		wg.Wait()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Len(t, result1, 3)
		assert.Len(t, result2, 3)

		for _, card := range cards {
			expected := "value_" + card.CardUUID
			if val, ok := result1[card.Title]; ok {
				assert.Equal(t, expected, val[0]["field"])
			}

			if val, ok := result2[card.Title]; ok {
				assert.Equal(t, expected, val[0]["field"])
			}
		}

		assert.LessOrEqual(t, maxRunning, 3, "Semaphore should enforce max parallelism")

		df.AssertExpectations(t)
	})
}

// Не parallel: сравнивает глобальный счётчик до/после.
func TestCollectErrorMetric(t *testing.T) {
	pg := &cmock.MockQueryFetcher{}
	pg.On("Fetch", mock.Anything, "analytics", "select 1", mock.Anything).
		Return(nil, errors.New("boom"))
	card := models.Card{
		Title: "c", CardUUID: "analytics", Type: models.CollectTypePostgres, Query: "select 1",
	}
	c := metrics.CollectorErrors.WithLabelValues("pg")
	before := testutil.ToFloat64(c)

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	col := collector.NewCollector(0, nil, nil, nil, pg, nil, nil, nil, l)
	_, err := col.Collect(t.Context(), card)

	require.Error(t, err)
	assert.InDelta(t, 1, testutil.ToFloat64(c)-before, 0)
}
