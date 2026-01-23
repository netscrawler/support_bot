package collector_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"support_bot/internal/collector"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cmock "support_bot/internal/collector/mock"
	models "support_bot/internal/models/report"
)

func TestCollect(t *testing.T) {
	t.Parallel()

	th := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := slog.New(th)

	t.Run("one query", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		df := &cmock.MockDataFetcher{}

		card := models.Card{Title: "card1", CardUUID: "uuid1"}

		df.On("Fetch", ctx, "uuid1").Return([]map[string]any{
			{"field": "value1"},
		}, nil)

		c := collector.NewCollector(0, df, l)

		result, err := c.Collect(ctx, card)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "value1", result["card1"][0]["field"])

		df.AssertExpectations(t)
	})

	t.Run("many query", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		df := &cmock.MockDataFetcher{}

		cards := []models.Card{
			{Title: "card1", CardUUID: "uuid1"},
			{Title: "card2", CardUUID: "uuid2"},
			{Title: "card3", CardUUID: "uuid3"},
		}

		for _, card := range cards {
			uuid := card.CardUUID
			df.On("Fetch", ctx, uuid).Run(func(_ mock.Arguments) {
				time.Sleep(1 * time.Second)
			}).Return([]map[string]any{{"field": "value_" + uuid}}, nil)
		}

		c := collector.NewCollector(3, df, l)

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
			{Title: "card1", CardUUID: "uuid1"},
			{Title: "card2", CardUUID: "uuid2"},
		}

		df.On("Fetch", ctx, "uuid1").Run(func(_ mock.Arguments) {
			time.Sleep(1 * time.Second)
		}).Return([]map[string]any{{"field": "value_" + "uuid1"}}, nil)

		df.On("Fetch", ctx, "uuid2").Run(func(_ mock.Arguments) {
			time.Sleep(1 * time.Second)
		}).Return(nil, errors.New("some error"))

		c := collector.NewCollector(3, df, l)

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
			{Title: "card1", CardUUID: "uuid1"},
			{Title: "card2", CardUUID: "uuid2"},
		}
		cards2 := []models.Card{
			{Title: "card3", CardUUID: "uuid3"},
			{Title: "card4", CardUUID: "uuid4"},
		}

		for _, card := range append(cards1, cards2...) {
			df.On("Fetch", ctx, card.CardUUID).Run(func(_ mock.Arguments) {
				time.Sleep(200 * time.Millisecond) // симуляция долгой работы
			}).Return([]map[string]any{{"field": "value_" + card.CardUUID}}, nil)
		}

		c := collector.NewCollector(2, df, slog.Default())

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
			{Title: "card1", CardUUID: "uuid1"},
			{Title: "card2", CardUUID: "uuid2"},
			{Title: "card3", CardUUID: "uuid3"},
			{Title: "card4", CardUUID: "uuid4"},
			{Title: "card5", CardUUID: "uuid5"},
			{Title: "card6", CardUUID: "uuid6"},
		}

		var mu sync.Mutex

		currentRunning := 0
		maxRunning := 0

		for _, card := range cards {
			uuid := card.CardUUID
			df.On("Fetch", ctx, uuid).Run(func(_ mock.Arguments) {
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

		c := collector.NewCollector(3, df, slog.Default())

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
