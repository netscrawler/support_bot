package pipeline_test

import (
	"context"
	"testing"
	"time"

	"support_bot/internal/metrics"
	"support_bot/internal/models"
	"support_bot/internal/processor/lua"
	luastd "support_bot/internal/processor/lua/stdlib"
	"support_bot/internal/processor/pipeline"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestLuaRunner_ErrorMetric(t *testing.T) {
	cfg := lua.Config{ExecutionTimeout: time.Minute, MaxMemoryMB: 64}
	runner := pipeline.NewLuaRunner(lua.NewManager(&cfg, nil, &luastd.STD{}))
	before := testutil.ToFloat64(
		metrics.LuaErrors.WithLabelValues("inline", "other"),
	)

	_, err := runner.Run(context.Background(), pipeline.Step{
		Script: `function process(params) error("boom") end`,
	}, models.Dataset{})

	require.Error(t, err)
	require.InDelta(
		t,
		before+1,
		testutil.ToFloat64(
			metrics.LuaErrors.WithLabelValues("inline", "other"),
		),
		0,
	)
}
