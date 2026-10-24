package stdlib

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"support_bot/internal/models"
	luapkg "support_bot/internal/pkg/lua"

	lua "github.com/yuin/gopher-lua"
)

// fnCollect — имя Lua-функции во всех модулях collector.*.
const (
	fnCollect = "collect"
)

type DirectCollector interface {
	Fetch(
		ctx context.Context,
		target string,
		params map[string]string,
	) ([]map[string]any, error)
}

// QueryCollector выполняет запрос в именованном источнике.
type QueryCollector interface {
	Fetch(
		ctx context.Context,
		target, query string,
		params map[string]string,
	) ([]map[string]any, error)
}

// CardCollector принимает карточку целиком (нужен для options).
type CardCollector interface {
	FetchCard(ctx context.Context, card models.Card) ([]map[string]any, error)
}

type CollectPlugin struct {
	directs map[string]DirectCollector
	queries map[string]QueryCollector
	http    CardCollector
}

// NewCollector создаёт коллектор; источник без настройки в queries не кладём.
func NewCollector(
	directs map[string]DirectCollector,
	queries map[string]QueryCollector,
) *CollectPlugin {
	if directs == nil {
		directs = make(map[string]DirectCollector)
	}

	return &CollectPlugin{directs: directs, queries: queries}
}

// WithHTTP включает collector.http.collect(target, query, params, options).
func (c *CollectPlugin) WithHTTP(h CardCollector) *CollectPlugin {
	c.http = h

	return c
}

func (c *CollectPlugin) Register(l *lua.LState) {
	root := l.NewTable()

	for name := range c.directs {
		mod := l.NewTable()

		l.SetFuncs(mod, map[string]lua.LGFunction{
			fnCollect: c.luaCollect(name),
		})

		root.RawSetString(name, mod)
	}

	for name, q := range c.queries {
		if q == nil {
			continue
		}

		mod := l.NewTable()
		l.SetFuncs(
			mod,
			map[string]lua.LGFunction{fnCollect: c.luaQueryCollect(name)},
		)
		root.RawSetString(name, mod)
	}

	if c.http != nil {
		mod := l.NewTable()
		l.SetFuncs(mod, map[string]lua.LGFunction{fnCollect: c.luaHTTPCollect})
		root.RawSetString("http", mod)
	}

	l.SetGlobal("collector", root)
}

func (c *CollectPlugin) luaCollect(name string) lua.LGFunction {
	return func(L *lua.LState) int {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		collector := c.directs[name]
		if collector == nil {
			L.Push(lua.LNil)
			L.Push(lua.LString("collector not found"))

			return 2
		}

		target := L.CheckString(1)

		rows, err := collector.Fetch(ctx, target, luaParams(L, 2))

		return pushRows(L, rows, err)
	}
}

func (c *CollectPlugin) luaQueryCollect(name string) lua.LGFunction {
	return func(l *lua.LState) int {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		target := l.CheckString(1)
		query := l.CheckString(2)

		rows, err := c.queries[name].Fetch(ctx, target, query, luaParams(l, 3))

		return pushRows(l, rows, err)
	}
}

// luaHTTPCollect: options — необязательная таблица, уходит в Card.Options.
// Ключ secret из Lua запрещён: секрет карточки задаётся только в БД.
func (c *CollectPlugin) luaHTTPCollect(l *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	card := models.Card{
		CardUUID: l.CheckString(1),
		Query:    l.CheckString(2),
		Params:   luaParams(l, 3),
	}

	if l.GetTop() >= 4 && l.Get(4) != lua.LNil {
		opts, ok := luapkg.LuaTableToGo(l.CheckTable(4)).(map[string]any)
		if !ok {
			return pushRows(
				l,
				nil,
				errors.New("options must be a table with string keys"),
			)
		}

		for k := range opts {
			if strings.EqualFold(k, "secret") {
				return pushRows(
					l,
					nil,
					errors.New("secret is not available from lua"),
				)
			}
		}

		raw, err := json.Marshal(opts)
		if err != nil {
			return pushRows(l, nil, err)
		}

		card.Options = raw
	}

	rows, err := c.http.FetchCard(ctx, card)

	return pushRows(l, rows, err)
}

// luaParams читает необязательную таблицу-аргумент idx в map строк.
func luaParams(l *lua.LState, idx int) map[string]string {
	params := map[string]string{}
	if l.GetTop() >= idx {
		l.CheckTable(idx).ForEach(func(k, v lua.LValue) {
			params[lua.LVAsString(k)] = lua.LVAsString(v)
		})
	}

	return params
}

func pushRows(l *lua.LState, rows []map[string]any, err error) int {
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(err.Error()))

		return 2
	}

	l.Push(rowsToLua(l, rows))
	l.Push(lua.LNil)

	return 2
}

func rowsToLua(
	l *lua.LState,
	rows []map[string]any,
) *lua.LTable {
	arr := l.NewTable()

	for _, row := range rows {
		rowTbl := l.NewTable()

		for k, v := range row {
			rowTbl.RawSetString(k, luapkg.GoValueToLua(l, v))
		}

		arr.Append(rowTbl)
	}

	return arr
}
