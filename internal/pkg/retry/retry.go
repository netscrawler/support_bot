package retry

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"time"

	"golang.org/x/time/rate"
)

var ErrQueueFull = errors.New("Queue full")

type Task struct {
	ID string

	Fn func(context.Context) error

	Attempt int
}

func NewTask(id string, fn func(context.Context) error) Task {
	return Task{
		ID:      id,
		Fn:      fn,
		Attempt: 0,
	}
}

type Policy interface {
	ShouldRetry(error) bool
}

type PolicyAlways struct{}

func (PolicyAlways) ShouldRetry(err error) bool {
	return true
}

type Backoff interface {
	Next(attempt int) time.Duration
}

type ExponentialBackoff struct {
	Base time.Duration
	Max  time.Duration
}

func (b ExponentialBackoff) Next(attempt int) time.Duration {
	delay := min(b.Base*time.Duration(1<<(attempt-1)), b.Max)

	delay = time.Duration(rand.Int64N(int64(delay)))

	return delay
}

type FixedBackoff struct {
	Delay time.Duration
}

func (b FixedBackoff) Next(int) time.Duration {
	return b.Delay
}

type Config struct {
	QueueSize  int
	Workers    int
	MaxRetries int

	Backoff Backoff
	Policy  Policy

	Logger *slog.Logger
	Silent bool

	RateLimit *rate.Limiter
}

type Retry struct {
	cfg *Config

	queue chan Task
}

func New(cfg Config) *Retry {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}

	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}

	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}

	if cfg.MaxRetries > 30 {
		cfg.MaxRetries = 30
	}

	if cfg.Backoff == nil {
		cfg.Backoff = &FixedBackoff{
			Delay: 5 * time.Second,
		}
	}

	if cfg.Policy == nil {
		cfg.Policy = PolicyAlways{}
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	if cfg.RateLimit == nil {
		cfg.RateLimit = rate.NewLimiter(rate.Limit(cfg.QueueSize), cfg.MaxRetries)
	}

	return &Retry{
		cfg: &cfg,

		queue: make(chan Task, cfg.QueueSize),
	}
}

func (r *Retry) Start(ctx context.Context) {
	for range r.cfg.Workers {
		go r.worker(ctx)
	}
}

func (r *Retry) Submit(task Task) error {
	select {
	case r.queue <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

func (r *Retry) AddRateLimit(rl *rate.Limiter) {
	r.cfg.RateLimit = rl
}

func (r *Retry) worker(ctx context.Context) {
	for {
		select {
		case task, ok := <-r.queue:
			if !ok {
				return
			}

			r.execute(ctx, task)

		case <-ctx.Done():
			if !r.cfg.Silent {
				r.cfg.Logger.InfoContext(ctx, "worker stopped")

				return
			}
		}
	}
}

func (r *Retry) execute(ctx context.Context, task Task) {
	if err := r.cfg.RateLimit.Wait(ctx); err != nil {
		if !r.cfg.Silent {
			r.cfg.Logger.InfoContext(ctx, "task failed", slog.Any("task id", task.ID))
		}

		return
	}

	err := task.Fn(ctx)
	if err == nil {
		return
	}

	if task.Attempt >= r.cfg.MaxRetries {
		if !r.cfg.Silent {
			r.cfg.Logger.InfoContext(
				ctx,
				"task failed by max retries",
				slog.Any("task id", task.ID),
				slog.Any("retries", task.Attempt),
				slog.Any("error", err),
			)
		}

		return
	}

	if !r.cfg.Policy.ShouldRetry(err) {
		if !r.cfg.Silent {
			r.cfg.Logger.InfoContext(
				ctx,
				"task failed by policy",
				slog.Any("task id", task.ID),
				slog.Any("retries", task.Attempt),
				slog.Any("error", err),
			)
		}

		return
	}

	task.Attempt++

	go func() {
		timer := time.NewTimer(r.cfg.Backoff.Next(task.Attempt))
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return

		case <-timer.C:
			err := r.Submit(task)
			if err != nil {
				if !r.cfg.Silent {
					r.cfg.Logger.InfoContext(ctx, "task failed (retrying)", slog.Any("error", err))
				}
			}
		}
	}()
}
