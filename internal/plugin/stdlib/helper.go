package stdlib

import (
	"reflect"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func luaTableToGoData(t *lua.LTable) (map[string][]map[string]any, error) {
	out := make(map[string][]map[string]any)

	t.ForEach(func(k, v lua.LValue) {
		key, ok := k.(lua.LString)
		if !ok {
			return
		}

		arr, ok := v.(*lua.LTable)
		if !ok {
			return
		}

		var rows []map[string]any

		arr.ForEach(func(_, row lua.LValue) {
			rowTbl, ok := row.(*lua.LTable)
			if !ok {
				return
			}

			m := make(map[string]any)
			rowTbl.ForEach(func(fk, fv lua.LValue) {
				m[fk.String()] = luaValueToGo(fv)
			})

			rows = append(rows, m)
		})

		out[key.String()] = rows
	})

	return out, nil
}

func luaValueToGo(v lua.LValue) any {
	switch x := v.(type) {
	case lua.LString:
		return string(x)
	case lua.LNumber:
		return float64(x)
	case lua.LBool:
		return bool(x)
	case *lua.LTable:
		return luaTableToGo(x)
	case *lua.LNilType:
		return nil
	default:
		return nil
	}
}

func luaTableToGo(t *lua.LTable) any {
	maxn := t.MaxN()
	
	if maxn > 0 {
		arr := make([]any, 0, maxn)
		for i := 1; i <= maxn; i++ {
			arr = append(arr, luaValueToGo(t.RawGetInt(i)))
		}
		return arr
	}
	
	m := make(map[string]any)
	t.ForEach(func(k, v lua.LValue) {
		if str, ok := k.(lua.LString); ok {
			m[string(str)] = luaValueToGo(v)
		}
	})
	return m
}

func goValueToLua(L *lua.LState, v any) lua.LValue {
	if v == nil {
		return lua.LNil
	}

	switch x := v.(type) {

	case string:
		return lua.LString(x)
	case bool:
		return lua.LBool(x)

	case int:
		return lua.LNumber(x)
	case int8:
		return lua.LNumber(x)
	case int16:
		return lua.LNumber(x)
	case int32:
		return lua.LNumber(x)
	case int64:
		return lua.LNumber(x)

	case uint:
		return lua.LNumber(x)
	case uint8:
		return lua.LNumber(x)
	case uint16:
		return lua.LNumber(x)
	case uint32:
		return lua.LNumber(x)
	case uint64:
		return lua.LNumber(x)

	case float32:
		return lua.LNumber(x)
	case float64:
		return lua.LNumber(x)

	case map[string]any:
		t := L.NewTable()
		for k, v := range x {
			t.RawSetString(k, goValueToLua(L, v))
		}
		return t

	case []any:
		t := L.NewTable()
		for _, v := range x {
			t.Append(goValueToLua(L, v))
		}
		return t

	default:
		return reflectToLua(L, x)
	}
}

func reflectToLua(L *lua.LState, v any) lua.LValue {
	rv := reflect.ValueOf(v)

	// pointer
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return lua.LNil
		}
		return reflectToLua(L, rv.Elem().Interface())
	}

	switch rv.Kind() {

	case reflect.Struct:
		t := L.NewTable()
		rt := rv.Type()

		for i := 0; i < rv.NumField(); i++ {
			field := rt.Field(i)

			if field.PkgPath != "" {
				continue
			}

			key := field.Name
			if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
				key = strings.Split(tag, ",")[0]
			}

			t.RawSetString(
				key,
				goValueToLua(L, rv.Field(i).Interface()),
			)
		}
		return t

	case reflect.Slice, reflect.Array:
		t := L.NewTable()
		for i := 0; i < rv.Len(); i++ {
			t.Append(goValueToLua(L, rv.Index(i).Interface()))
		}
		return t

	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return lua.LNil
		}

		t := L.NewTable()
		for _, key := range rv.MapKeys() {
			t.RawSetString(
				key.String(),
				goValueToLua(L, rv.MapIndex(key).Interface()),
			)
		}
		return t

	default:
		return lua.LNil
	}
}
