package stdlib

import (
	"context"
	"errors"
	"time"

	models "support_bot/internal/models/report"

	lua "github.com/yuin/gopher-lua"
)

type Collector interface {
	Collect(ctx context.Context, cards ...models.Card) (map[string][]map[string]any, error)
}

type CollectPlugin struct {
	collct Collector
}

func NewCollector(collct Collector) *CollectPlugin {
	return &CollectPlugin{collct: collct}
}

func (c *CollectPlugin) luaCollect(L *lua.LState) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	luaCards := L.CheckTable(1)

	cards := make([]models.Card, 0, luaCards.Len())
	var convErr error

	luaCards.ForEach(func(_, v lua.LValue) {
		if convErr != nil {
			return
		}

		tbl, ok := v.(*lua.LTable)
		if !ok {
			convErr = errors.New("each card must be table")
			return
		}

		card, err := cardFromLua(tbl)
		if err != nil {
			convErr = err
			return
		}

		cards = append(cards, card)
	})

	if convErr != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(convErr.Error()))
		return 2
	}

	res, err := c.collct.Collect(ctx, cards...)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(mapResultToLua(L, res))
	L.Push(lua.LNil)
	return 2
}

func mapResultToLua(
	L *lua.LState,
	data map[string][]map[string]any,
) *lua.LTable {
	root := L.NewTable()

	for name, rows := range data {
		arr := L.NewTable()

		for _, row := range rows {
			rowTbl := L.NewTable()
			for k, v := range row {
				rowTbl.RawSetString(k, goValueToLua(L, v))
			}
			arr.Append(rowTbl)
		}

		root.RawSetString(name, arr)
	}

	return root
}

func cardFromLua(t *lua.LTable) (models.Card, error) {
	uuid := t.RawGetString("card_uuid")
	if uuid.Type() != lua.LTString {
		return models.Card{}, errors.New("card.card_uuid must be string")
	}

	title := t.RawGetString("title")
	if title.Type() != lua.LTString {
		return models.Card{}, errors.New("card.title must be string")
	}

	return models.Card{
		CardUUID: uuid.String(),
		Title:    title.String(),
	}, nil
}
