package processor_test

import (
	"context"
	"log/slog"
	"support_bot/internal/models"
	"support_bot/internal/processor"
	"support_bot/internal/processor/lua"
	"support_bot/internal/processor/pipeline"
	"testing"
	"time"

	luastd "support_bot/internal/processor/lua/stdlib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessor_Process_WithLua - интеграционные тесты процессора с реальным Lua
func TestProcessor_Process_WithLua(t *testing.T) {
	l := slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := lua.Config{
		ExecutionTimeout: 5 * time.Minute,
		MaxMemoryMB:      256,
		AllowedModules: []string{
			"json",
			"http",
			"url",
			"time",
			"strings",
			"inspect",
		},
	}

	mng := lua.NewManager(&cfg, nil, luastd.NewSTD(luastd.RateLimit{}, luastd.DatabasePlugin{}))
	luaRunner := pipeline.NewLuaRunner(mng)

	tests := []struct {
		name       string
		data       models.Dataset
		pipeline   *pipeline.Pipeline
		expected   models.Dataset
		wantErr    bool
		errContain string
	}{
		{
			name: "lua pipeline adds fields to data",
			data: models.Dataset{
				"users": []map[string]any{
					{"id": 1, "name": "Alice"},
					{"id": 2, "name": "Bob"},
				},
			},
			pipeline: &pipeline.Pipeline{
				Name: "add_fields",
				Steps: []models.Step{
					{
						ID:   "step1",
						Type: "lua",
						Script: `
function process(params)
    local result = {}
    for key, list in pairs(params) do
        result[key] = {}
        for i, item in ipairs(list) do
            local new_item = {}
            for k, v in pairs(item) do
                new_item[k] = v
            end
            new_item.processed = true
            table.insert(result[key], new_item)
        end
    end
    return result
end
`,
					},
				},
			},
			expected: models.Dataset{
				"users": []map[string]any{
					{
						"id":        int64(1),
						"name":      "Alice",
						"processed": true,
					},
					{
						"id":        int64(2),
						"name":      "Bob",
						"processed": true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "lua pipeline with multiple sequential steps",
			data: models.Dataset{
				"data": []map[string]any{
					{"value": 10},
					{"value": 20},
				},
			},
			pipeline: &pipeline.Pipeline{
				Name: "multi_step_pipeline",
				Steps: []models.Step{
					{
						ID:   "step1",
						Type: "lua",
						Script: `
function process(params)
    local result = {}
    for key, list in pairs(params) do
        result[key] = {}
        for i, item in ipairs(list) do
            local new_item = {}
            for k, v in pairs(item) do
                new_item[k] = v
            end
            new_item.doubled = new_item.value * 2
            table.insert(result[key], new_item)
        end
    end
    return result
end
`,
					},
					{
						ID:   "step2",
						Type: "lua",
						Script: `
function process(params)
    local result = {}
    for key, list in pairs(params) do
        result[key] = {}
        for i, item in ipairs(list) do
            local new_item = {}
            for k, v in pairs(item) do
                new_item[k] = v
            end
            new_item.tripled = new_item.doubled * 1.5
            table.insert(result[key], new_item)
        end
    end
    return result
end
`,
					},
				},
			},
			expected: models.Dataset{
				"data": []map[string]any{
					{
						"value":   int64(10),
						"doubled": int64(20),
						"tripled": int64(30),
					},
					{
						"value":   int64(20),
						"doubled": int64(40),
						"tripled": int64(60),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "lua pipeline filters data",
			data: models.Dataset{
				"users": []map[string]any{
					{"id": 1, "name": "Alice", "age": 25},
					{"id": 2, "name": "Bob", "age": 17},
					{"id": 3, "name": "Charlie", "age": 30},
				},
			},
			pipeline: &pipeline.Pipeline{
				Name: "filter_adults",
				Steps: []models.Step{
					{
						ID:   "step1",
						Type: "lua",
						Script: `
function process(params)
    local result = {}
    for key, list in pairs(params) do
        result[key] = {}
        for i, item in ipairs(list) do
            if item.age >= 18 then
                table.insert(result[key], item)
            end
        end
    end
    return result
end
`,
					},
				},
			},
			expected: models.Dataset{
				"users": []map[string]any{
					{"id": int64(1), "name": "Alice", "age": int64(25)},
					{"id": int64(3), "name": "Charlie", "age": int64(30)},
				},
			},
			wantErr: false,
		},
		{
			name: "lua pipeline with JSON operations",
			data: models.Dataset{
				"records": []map[string]any{
					{"id": 1, "data_json": `{"key":"value","number":42}`},
				},
			},
			pipeline: &pipeline.Pipeline{
				Name: "parse_json",
				Steps: []models.Step{
					{
						ID:   "step1",
						Type: "lua",
						Script: `
function process(params)
    local json = require("json")
    local result = {}
    for key, list in pairs(params) do
        result[key] = {}
        for i, item in ipairs(list) do
            local new_item = {}
            for k, v in pairs(item) do
                new_item[k] = v
            end
            if item.data_json then
                new_item.parsed = json.decode(item.data_json)
            end
            table.insert(result[key], new_item)
        end
    end
    return result
end
`,
					},
				},
			},
			expected: models.Dataset{
				"records": []map[string]any{
					{
						"id":        int64(1),
						"data_json": `{"key":"value","number":42}`,
						"parsed":    map[string]any{"key": "value", "number": int64(42)},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty pipeline with data",
			data: models.Dataset{
				"test": []map[string]any{
					{"value": 1},
				},
			},
			pipeline: &pipeline.Pipeline{
				Name:  "empty",
				Steps: []models.Step{},
			},
			expected: models.Dataset{
				"test": []map[string]any{
					{
						"value": 1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "lua pipeline with string operations",
			data: models.Dataset{
				"names": []map[string]any{
					{"name": "alice"},
					{"name": "bob"},
				},
			},
			pipeline: &pipeline.Pipeline{
				Name: "uppercase_names",
				Steps: []models.Step{
					{
						ID:   "step1",
						Type: "lua",
						Script: `
function process(params)
    local result = {}

    for key, items in pairs(params) do
        result[key] = {}

        for _, item in ipairs(items) do
            local newItem = {}

            for k, v in pairs(item) do
                newItem[k] = v
            end

            if type(item.name) == "string" then
                newItem.upper_name = string.upper(item.name)
            end

            table.insert(result[key], newItem)
        end
    end

    return result
end`,
					},
				},
			},
			expected: models.Dataset{
				"names": []map[string]any{
					{
						"name":       "alice",
						"upper_name": "ALICE",
					},
					{
						"name":       "bob",
						"upper_name": "BOB",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "context cancellation stops pipeline",
			data: models.Dataset{
				"test": []map[string]any{{"value": 1}},
			},
			pipeline: &pipeline.Pipeline{
				Name: "should_cancel",
				Steps: []models.Step{
					{
						ID:     "step1",
						Type:   "lua",
						Script: "function process(p) return p end",
					},
				},
			},
			wantErr:    true,
			expected:   nil,
			errContain: "stopped on step",
		},
		{
			name: "invalid lua script returns error",
			data: models.Dataset{
				"test": []map[string]any{{"value": 1}},
			},
			pipeline: &pipeline.Pipeline{
				Name: "invalid_lua",
				Steps: []models.Step{
					{
						ID:     "step1",
						Type:   "lua",
						Script: "this is not valid lua @#$%",
					},
				},
			},
			wantErr:    true,
			expected:   nil,
			errContain: "failed",
		},
		{
			name: "missing runner type returns error",
			data: models.Dataset{
				"test": []map[string]any{{"value": 1}},
			},
			pipeline: &pipeline.Pipeline{
				Name: "missing_type",
				Steps: []models.Step{
					{
						ID:   "step1",
						Type: "nonexistent_runner",
					},
				},
			},
			wantErr:    true,
			expected:   nil,
			errContain: "runner not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.name == "context cancellation stops pipeline" {
				ctxCancel, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = ctxCancel
			} else {
				ctx = context.Background()
			}

			rg := processor.NewReg()
			rg.Register("lua", luaRunner)
			p := processor.NewProcessor(rg, l)

			got, err := p.Process(ctx, tt.data, tt.pipeline)

			assert.Equal(t, tt.expected, got, "unexpected result dataset")
			if tt.wantErr {
				require.Error(t, err, "expected error but got nil")
				if tt.errContain != "" {
					require.ErrorContains(
						t,
						err,
						tt.errContain,
						"error message should contain expected text",
					)
				}
			} else {
				require.NoError(t, err, "unexpected error")
				require.NotNil(t, got, "expected non-nil result")
			}
		})
	}
}

func TestRunnerRegistry(t *testing.T) {
	t.Run("register and get runner", func(t *testing.T) {
		rg := processor.NewReg()

		cfg := lua.Config{
			ExecutionTimeout: 5 * time.Minute,
			MaxMemoryMB:      256,
			AllowedModules:   []string{},
		}
		mng := lua.NewManager(&cfg, nil, &luastd.STD{})
		luaRunner := pipeline.NewLuaRunner(mng)

		rg.Register("lua", luaRunner)
		got, err := rg.Get("lua")

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, luaRunner, got)
	})

	t.Run("case insensitive lookup", func(t *testing.T) {
		rg := processor.NewReg()

		cfg := lua.Config{
			ExecutionTimeout: 5 * time.Minute,
			MaxMemoryMB:      256,
			AllowedModules:   []string{},
		}
		mng := lua.NewManager(&cfg, nil, &luastd.STD{})
		luaRunner := pipeline.NewLuaRunner(mng)

		rg.Register("LuaRunner", luaRunner)
		got, err := rg.Get("luarunner")

		require.NoError(t, err)
		require.Equal(t, luaRunner, got)
	})

	t.Run("get non-existent runner returns error", func(t *testing.T) {
		rg := processor.NewReg()
		_, err := rg.Get("nonexistent")

		require.Error(t, err)
		require.Contains(t, err.Error(), "runner not found")
	})

	t.Run("overwrite existing runner", func(t *testing.T) {
		rg := processor.NewReg()

		cfg := lua.Config{
			ExecutionTimeout: 5 * time.Minute,
			MaxMemoryMB:      256,
			AllowedModules:   []string{},
		}
		mng1 := lua.NewManager(&cfg, nil, &luastd.STD{})
		mng2 := lua.NewManager(&cfg, nil, &luastd.STD{})
		luaRunner1 := pipeline.NewLuaRunner(mng1)
		luaRunner2 := pipeline.NewLuaRunner(mng2)

		rg.Register("lua", luaRunner1)
		rg.Register("lua", luaRunner2)
		got, _ := rg.Get("lua")

		require.Equal(t, luaRunner2, got)
	})
}
