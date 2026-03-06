package plugins_test

import (
	"encoding/json"
	"testing"

	"support_bot/internal/plugin/stdlib"

	plugins "support_bot/internal/plugin"

	"github.com/stretchr/testify/require"
)

func TestLuaPlugin_Execute(t *testing.T) {
	t.Parallel()

	t.Run("extend data", func(t *testing.T) {
		plug := `
-- Пример Lua плагина с использованием stdlib

plugin = {
	name = "example_with_stdlib",
	version = "1.0.0",
	description = "Пример плагина с использованием стандартной библиотеки",
	author = "Your Name",

	-- Инициализация плагина
	init = function(config)
		-- Сохраняем конфигурацию
		plugin.config = config
		return true, nil
	end,

	-- Валидация параметров
	validate = function(params)
		return true, nil
	end,

	-- на вход подается map[string][]map[string]any из go как обогатить map[string]any и вернуть новое значение
	execute = function(params)
		local result = {}

		for key, list in pairs(params) do
			result[key] = {}

			for i, item in ipairs(list) do
				local new_item = {
					Phone = item.Phone,
					Name  = item.Name,
					Data  = item.Data,
				}

				new_item.processed = true
				new_item.index = i
				new_item.source = key
				new_item.processed_at = os.time()

				table.insert(result[key], new_item)
			end
		end
		return result
	end,

	-- Очистка ресурсов
	cleanup = function()
		plugin.config = nil
	end
}
`

		plugin, err := plugins.NewLuaPluginWithConfigFromString(
			plug,
			plugins.DefaultRuntimeConfig(),
			&stdlib.STD{},
		)
		require.NoError(t, err)

		err = plugin.Init(nil)
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

		params := map[string]any{
			"Users": users["Users"],
		}

		// func (p *LuaPlugin) Execute(_ context.Context, params map[string]any) ([]byte, error) {

		data, err := plugin.Execute(t.Context(), params)
		require.NoError(t, err)
		t.Log(string(data))

		var newData map[string][]map[string]any

		err = json.Unmarshal(data, &newData)
		require.NoError(t, err)
		t.Log(newData)
	})

	t.Run("extend data with fetch data", func(t *testing.T) {
		plug := `
-- Пример Lua плагина с использованием stdlib
--
function getOperatorFromNumber(number)
	local http = require("http")
	local json = require("json")

	-- убираем первую 7
	local trimmedNumber = tostring(number):sub(2)

	-- формируем URL с query
	local url = "https://num.voxlink.ru/get/?num=" .. trimmedNumber

	-- делаем запрос
	local response, err = http.request("GET", url)
	if not response then
		return nil, err
	end

	if response.status_code ~= 200 then
		return nil, "bad status: " .. response.status_code
	end

	local data, _, parseErr = json.decode(response.body, 1, nil)
	if parseErr then
		return nil, parseErr
	end

	return data.operator
end

plugin = {
	name = "example_with_stdlib",
	version = "1.0.0",
	description = "Пример плагина с использованием стандартной библиотеки",
	author = "Your Name",

	-- Инициализация плагина
	init = function(config)
		-- Сохраняем конфигурацию
		plugin.config = config
		return true, nil
	end,

	-- Валидация параметров
	validate = function(params)
		return true, nil
	end,

	-- на вход подается map[string][]map[string]any из go как обогатить map[string]any и вернуть новое значение
	execute = function(params)
		local result = {}

		for key, list in pairs(params) do
			result[key] = {}

			for i, item in ipairs(list) do
				local new_item = {
					Phone = item.Phone,
					Name = item.Name,
					Data = item.Data,
				}

				new_item.operator = getOperatorFromNumber(tostring(new_item.Phone))
				new_item.processed = true
				new_item.index = i
				new_item.source = key
				new_item.processed_at = os.time()

				table.insert(result[key], new_item)
			end
		end
		return result
	end,

	-- Очистка ресурсов
	cleanup = function()
		plugin.config = nil
	end
}
`

		plugin, err := plugins.NewLuaPluginWithConfigFromString(
			plug,
			plugins.DefaultRuntimeConfig(),
			&stdlib.STD{},
		)
		require.NoError(t, err)

		err = plugin.Init(nil)
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

		params := map[string]any{
			"Users": users["Users"],
		}

		// func (p *LuaPlugin) Execute(_ context.Context, params map[string]any) ([]byte, error) {

		data, err := plugin.Execute(t.Context(), params)
		require.NoError(t, err)
		t.Log(string(data))

		var newData map[string][]map[string]any

		err = json.Unmarshal(data, &newData)
		require.NoError(t, err)
		t.Log(newData)
	})
}
