package lua_test

import (
	"testing"

	plugins "support_bot/internal/processor/lua"
	"support_bot/internal/processor/lua/stdlib"

	"github.com/stretchr/testify/require"
)

func TestLuaPlugin_Execute(t *testing.T) {
	t.Parallel()

	t.Run("extend data", func(t *testing.T) {
		plug := `
-- Пример Lua плагина с использованием stdlib
function process(params)
    local result = {}

    for key, list in pairs(params) do
        result[key] = {}

        for i, item in ipairs(list) do
            local new_item = {
                Phone = item.Phone,
                Name = item.Name,
                Data = item.Data,
            }

            new_item.processed = true
            new_item.index = i
            new_item.source = key
            new_item.processed_at = os.time()

            table.insert(result[key], new_item)
        end
    end

    return result
end
`

		plugin, err := plugins.NewLuaPlugin(
			plug,
			plugins.DefaultRuntimeConfig(),
			&stdlib.STD{},
		)
		require.NoError(t, err)

		users := map[string][]map[string]any{
			"Users": {
				map[string]any{
					"Phone": 79097187978,
					"Name":  "Ivan",
					"Data":  "some_data",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
			},
		}

		data, err := plugin.Execute(t.Context(), users)
		require.NoError(t, err)
		t.Log(data)
	})

	t.Run("extend data with fetch data", func(t *testing.T) {
		plug := `
		-- Пример Lua плагина с использованием stdlib
function getOperatorFromNumber(number)
    local http = require("http")
    local json = require("json")
    local response, err = http.request("GET", "https://num.voxlink.ru/get/", {
        query = "num=" .. tostring(number):sub(2),
        headers = { ["Accept"] = "application/json" },
        timeout = "30s",
    })
    if not response then
        return nil, err
    end
    if response.status_code ~= 200 then
        return nil, "bad status " .. response.status_code
    end
    local data, _, parseErr = json.decode(response.body, 1, nil)
    if parseErr then
        return nil, parseErr
    end
    return data.operator
end

function process(params)
    local result = {}

    local rl = rate.new(9,9)

    for key, list in pairs(params) do
        result[key] = {}

        for i, item in ipairs(list) do
            local new_item = {
                Phone = item.Phone,
                Name = item.Name,
                Data = item.Data,
            }

            local err = rl:wait()
            if err then
                new_item.error = err
            end
            new_item.operator = getOperatorFromNumber(new_item.Phone)
            new_item.processed = true
            new_item.index = i
            new_item.source = key
            new_item.processed_at = os.time()

            table.insert(result[key], new_item)
        end
    end

    return result
end
`

		std := stdlib.NewSTD(stdlib.RateLimit{})
		plugin, err := plugins.NewLuaPlugin(
			plug,
			plugins.DefaultRuntimeConfig(),
			std,
		)
		require.NoError(t, err)

		users := map[string][]map[string]any{
			"Users": {
				map[string]any{
					"Phone": 79097187978,
					"Name":  "Ivan",
					"Data":  "some_data",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
				map[string]any{
					"Phone": 79502323122,
					"Name":  "Petr",
					"Data":  "some_data2",
				},
			},
		}

		data, err := plugin.Execute(t.Context(), users)
		require.NoError(t, err)
		t.Log(data)
	})
}
