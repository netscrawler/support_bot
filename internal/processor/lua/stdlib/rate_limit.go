package stdlib

import (
	"context"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/time/rate"
)

const rateLimitType = "RateLimit"

type RateLimit struct{}

func (r RateLimit) Register(L *lua.LState) {
	mt := L.NewTypeMetatable(rateLimitType)

	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"allow": r.luaAllow,
		"wait":  r.luaWait,
	}))

	mod := L.NewTable()
	L.SetFuncs(mod, map[string]lua.LGFunction{
		"new": r.luaNew,
	})

	L.SetGlobal("rate", mod)
}

func (r *RateLimit) luaNew(L *lua.LState) int {
	limit := rate.Limit(L.CheckNumber(1))
	burst := L.CheckInt(2)

	ud := L.NewUserData()
	ud.Value = rate.NewLimiter(limit, burst)

	L.SetMetatable(ud, L.GetTypeMetatable(rateLimitType))

	L.Push(ud)
	return 1
}

func (r *RateLimit) luaAllow(L *lua.LState) int {
	ud := L.CheckUserData(1)

	rl, ok := ud.Value.(*rate.Limiter)
	if !ok {
		L.ArgError(1, "RateLimit expected")
		return 0
	}

	L.Push(lua.LBool(rl.Allow()))
	return 1
}

func (r *RateLimit) luaWait(L *lua.LState) int {
	ud := L.CheckUserData(1)

	rl, ok := ud.Value.(*rate.Limiter)
	if !ok {
		L.ArgError(1, "RateLimit expected")

		return 0
	}

	if err := rl.Wait(context.Background()); err != nil {
		L.Push(lua.LString(err.Error()))

		return 1
	}

	L.Push(lua.LNil)

	return 1
}
