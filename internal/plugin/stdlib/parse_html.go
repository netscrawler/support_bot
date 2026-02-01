package stdlib

import (
	"support_bot/internal/pkg/text"

	lua "github.com/yuin/gopher-lua"
)

func luaExecuteTemplate(L *lua.LState) int {
	// --- args ---
	templ := L.CheckString(1)
	dataTable := L.OptTable(2, nil)

	// --- Lua → Go ---
	var data any
	if dataTable != nil {
		data = luaValueToGo(dataTable)
	}

	// --- вызов ExecuteTemplate ---
	result, err := text.ExecuteTemplate(templ, data)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	// --- Go → Lua ---
	L.Push(lua.LString(result))
	L.Push(lua.LNil)
	return 2
}
