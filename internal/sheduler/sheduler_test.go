package sheduler

import (
	"log/slog"
	"testing"

	"support_bot/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestStopResetsRunningGauge(t *testing.T) {
	metrics.SchedulerRunning.Set(1)

	NewSheduler(nil, slog.Default(), nil, nil).Stop()

	assert.InDelta(t, 0, testutil.ToFloat64(metrics.SchedulerRunning), 0)
}
