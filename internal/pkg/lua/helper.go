package luapkg

import (
	"math"
	"reflect"
	"strings"

	luaEngine "github.com/yuin/gopher-lua"
)

func LuaTableToGoData(t *luaEngine.LTable) (map[string][]map[string]any, error) {
	out := make(map[string][]map[string]any)

	t.ForEach(func(k, v luaEngine.LValue) {
		key, ok := k.(luaEngine.LString)
		if !ok {
			return
		}

		arr, ok := v.(*luaEngine.LTable)
		if !ok {
			return
		}

		var rows []map[string]any

		arr.ForEach(func(_, row luaEngine.LValue) {
			rowTbl, ok := row.(*luaEngine.LTable)
			if !ok {
				return
			}

			m := make(map[string]any)

			rowTbl.ForEach(func(fk, fv luaEngine.LValue) {
				m[fk.String()] = LuaValueToGo(fv)
			})

			rows = append(rows, m)
		})

		out[key.String()] = rows
	})

	return out, nil
}

func LuaValueToGo(v luaEngine.LValue) any {
	switch x := v.(type) {
	case luaEngine.LString:
		return string(x)
	case luaEngine.LNumber:
		f := float64(x)

		if math.Trunc(f) != f {
			return f
		}
		if f >= math.MinInt64 && f <= math.MaxInt64 {
			return int64(f)
		}

		return f
	case luaEngine.LBool:
		return bool(x)
	case *luaEngine.LTable:
		return LuaTableToGo(x)
	case *luaEngine.LNilType:
		return nil
	default:
		return nil
	}
}

func LuaTableToGo(t *luaEngine.LTable) any {
	maxn := t.MaxN()

	if maxn > 0 {
		arr := make([]any, 0, maxn)
		for i := 1; i <= maxn; i++ {
			arr = append(arr, LuaValueToGo(t.RawGetInt(i)))
		}

		return arr
	}

	m := make(map[string]any)

	t.ForEach(func(k, v luaEngine.LValue) {
		if str, ok := k.(luaEngine.LString); ok {
			m[string(str)] = LuaValueToGo(v)
		}
	})

	return m
}

func GoValueToLua(L *luaEngine.LState, v any) luaEngine.LValue {
	if v == nil {
		return luaEngine.LNil
	}

	switch x := v.(type) {
	case string:
		return luaEngine.LString(x)
	case bool:
		return luaEngine.LBool(x)

	case int:
		return luaEngine.LNumber(x)
	case int8:
		return luaEngine.LNumber(x)
	case int16:
		return luaEngine.LNumber(x)
	case int32:
		return luaEngine.LNumber(x)
	case int64:
		return luaEngine.LNumber(x)

	case uint:
		return luaEngine.LNumber(x)
	case uint8:
		return luaEngine.LNumber(x)
	case uint16:
		return luaEngine.LNumber(x)
	case uint32:
		return luaEngine.LNumber(x)
	case uint64:
		return luaEngine.LNumber(x)

	case float32:
		return luaEngine.LNumber(x)
	case float64:
		return luaEngine.LNumber(x)

	case map[string]any:
		t := L.NewTable()

		for k, v := range x {
			t.RawSetString(k, GoValueToLua(L, v))
		}

		return t

	case []any:
		t := L.NewTable()

		for _, v := range x {
			t.Append(GoValueToLua(L, v))
		}

		return t

	default:
		return ReflectToLua(L, x)
	}
}

func ReflectToLua(L *luaEngine.LState, v any) luaEngine.LValue {
	rv := reflect.ValueOf(v)

	// pointer
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return luaEngine.LNil
		}

		return ReflectToLua(L, rv.Elem().Interface())
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
				GoValueToLua(L, rv.Field(i).Interface()),
			)
		}

		return t

	case reflect.Slice, reflect.Array:
		t := L.NewTable()

		for i := 0; i < rv.Len(); i++ {
			t.Append(GoValueToLua(L, rv.Index(i).Interface()))
		}

		return t

	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return luaEngine.LNil
		}

		t := L.NewTable()

		for _, key := range rv.MapKeys() {
			t.RawSetString(
				key.String(),
				GoValueToLua(L, rv.MapIndex(key).Interface()),
			)
		}

		return t

	default:
		return luaEngine.LNil
	}
}

func MapResultToLua(
	L *luaEngine.LState,
	data map[string][]map[string]any,
) *luaEngine.LTable {
	root := L.NewTable()

	for name, rows := range data {
		arr := L.NewTable()

		for _, row := range rows {
			rowTbl := L.NewTable()

			for k, v := range row {
				rowTbl.RawSetString(k, GoValueToLua(L, v))
			}

			arr.Append(rowTbl)
		}

		root.RawSetString(name, arr)
	}

	return root
}
