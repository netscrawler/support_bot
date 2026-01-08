package collector_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"support_bot/internal/collector"
	cmock "support_bot/internal/collector/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCollect(t *testing.T) {
	t.Parallel()
	t.Run("one query", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()

		df := &cmock.MockDataFetcher{}

		card := collector.Card{Name: "card1", UUID: "uuid1"}

		df.On("Fetch", ctx, "uuid1").Return([]map[string]any{
			{"field": "value1"},
		}, nil)

		c := collector.NewCollector(0, df, slog.Default())

		result, err := c.Collect(ctx, card)

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "value1", result["card1"][0]["field"])

		df.AssertExpectations(t)
	})

	t.Run("many query", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		df := &cmock.MockDataFetcher{}

		cards := []collector.Card{
			{Name: "card1", UUID: "uuid1"},
			{Name: "card2", UUID: "uuid2"},
			{Name: "card3", UUID: "uuid3"},
		}

		for _, card := range cards {
			uuid := card.UUID
			df.On("Fetch", ctx, uuid).Run(func(args mock.Arguments) {
				time.Sleep(1 * time.Second)
			}).Return([]map[string]any{{"field": "value_" + uuid}}, nil)
		}

		c := collector.NewCollector(3, df, slog.Default())

		start := time.Now()
		result, err := c.Collect(ctx, cards...)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Len(t, result, 3)

		for _, card := range cards {
			assert.Equal(t, "value_"+card.UUID, result[card.Name][0]["field"])
		}

		df.AssertExpectations(t)

		assert.Less(t, duration.Seconds(), 1.5, "Tasks should run in parallel")
	})

	t.Run("many query with one err", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		df := &cmock.MockDataFetcher{}

		cards := []collector.Card{
			{Name: "card1", UUID: "uuid1"},
			{Name: "card2", UUID: "uuid2"},
		}

		df.On("Fetch", ctx, "uuid1").Run(func(args mock.Arguments) {
			time.Sleep(1 * time.Second)
		}).Return([]map[string]any{{"field": "value_" + "uuid1"}}, nil)

		df.On("Fetch", ctx, "uuid2").Run(func(args mock.Arguments) {
			time.Sleep(1 * time.Second)
		}).Return(nil, errors.New("some error"))

		c := collector.NewCollector(3, df, slog.Default())

		start := time.Now()
		result, err := c.Collect(ctx, cards...)
		duration := time.Since(start)

		assert.EqualError(t, err, "some error")
		assert.Len(t, result, 2)

		assert.Equal(t, "value_uuid1", result["card1"][0]["field"])
		assert.Equal(t, 0, len(result["card2"]))

		df.AssertExpectations(t)

		assert.Less(t, duration.Seconds(), 1.5, "Tasks should run in parallel")
	})
	t.Run("single collector multiple goroutines", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		df := &cmock.MockDataFetcher{}

		cards1 := []collector.Card{
			{Name: "card1", UUID: "uuid1"},
			{Name: "card2", UUID: "uuid2"},
		}
		cards2 := []collector.Card{
			{Name: "card3", UUID: "uuid3"},
			{Name: "card4", UUID: "uuid4"},
		}

		for _, card := range append(cards1, cards2...) {
			df.On("Fetch", ctx, card.UUID).Run(func(args mock.Arguments) {
				time.Sleep(200 * time.Millisecond) // симуляция долгой работы
			}).Return([]map[string]any{{"field": "value_" + card.UUID}}, nil)
		}

		c := collector.NewCollector(2, df, slog.Default())

		start := time.Now()

		var wg sync.WaitGroup
		wg.Add(2)

		var result1, result2 map[string][]map[string]any
		var err1, err2 error

		go func() {
			defer wg.Done()
			result1, err1 = c.Collect(ctx, cards1...)
		}()

		go func() {
			defer wg.Done()
			result2, err2 = c.Collect(ctx, cards2...)
		}()

		wg.Wait()
		duration := time.Since(start)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Len(t, result1, 2)
		assert.Len(t, result2, 2)

		for _, card := range cards1 {
			expected := "value_" + card.UUID
			assert.Equal(
				t,
				expected,
				result1[card.Name][0]["field"],
				"card in result1",
			)
		}
		for _, card := range cards2 {
			expected := "value_" + card.UUID
			assert.Equal(
				t,
				expected,
				result2[card.Name][0]["field"],
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

		// Создаём 6 карт
		cards := []collector.Card{
			{Name: "card1", UUID: "uuid1"},
			{Name: "card2", UUID: "uuid2"},
			{Name: "card3", UUID: "uuid3"},
			{Name: "card4", UUID: "uuid4"},
			{Name: "card5", UUID: "uuid5"},
			{Name: "card6", UUID: "uuid6"},
		}

		var mu sync.Mutex
		currentRunning := 0
		maxRunning := 0

		for _, card := range cards {
			uuid := card.UUID
			df.On("Fetch", ctx, uuid).Run(func(args mock.Arguments) {
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
		wg.Add(2)

		var result1, result2 map[string][]map[string]any
		var err1, err2 error

		go func() {
			defer wg.Done()
			result1, err1 = c.Collect(ctx, cards[:3]...)
		}()

		go func() {
			defer wg.Done()
			result2, err2 = c.Collect(ctx, cards[3:]...)
		}()

		wg.Wait()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Len(t, result1, 3)
		assert.Len(t, result2, 3)

		for _, card := range cards {
			expected := "value_" + card.UUID
			if val, ok := result1[card.Name]; ok {
				assert.Equal(t, expected, val[0]["field"])
			}
			if val, ok := result2[card.Name]; ok {
				assert.Equal(t, expected, val[0]["field"])
			}
		}

		assert.LessOrEqual(t, maxRunning, 3, "Semaphore should enforce max parallelism")

		df.AssertExpectations(t)
	})
}
