package stdlib

import (
	"context"

	lua "github.com/yuin/gopher-lua"
	"golang.org/x/time/rate"
)

const (
	rateLimitType = "RateLimit"
)

type RateLimit struct{}

func (r RateLimit) Register(l *lua.LState) {
	mt := l.NewTypeMetatable(rateLimitType)

	l.SetField(
		mt,
		"__index",
		l.SetFuncs(l.NewTable(), map[string]lua.LGFunction{
			"allow": r.luaAllow,
			"wait":  r.luaWait,
		}),
	)

	mod := l.NewTable()
	l.SetFuncs(mod, map[string]lua.LGFunction{
		"new": r.luaNew,
	})

	l.SetGlobal("rate", mod)
}

func (r RateLimit) luaNew(l *lua.LState) int {
	limit := rate.Limit(l.CheckNumber(1))
	burst := l.CheckInt(2)

	ud := l.NewUserData()

	ud.Value = rate.NewLimiter(limit, burst)

	l.SetMetatable(ud, l.GetTypeMetatable(rateLimitType))

	l.Push(ud)

	return 1
}

func (r RateLimit) luaAllow(l *lua.LState) int {
	ud := l.CheckUserData(1)

	rl, ok := ud.Value.(*rate.Limiter)
	if !ok {
		l.ArgError(1, "RateLimit expected")

		return 0
	}

	l.Push(lua.LBool(rl.Allow()))

	return 1
}

func (r RateLimit) luaWait(l *lua.LState) int {
	ud := l.CheckUserData(1)

	rl, ok := ud.Value.(*rate.Limiter)
	if !ok {
		l.ArgError(1, "RateLimit expected")

		return 0
	}

	if err := rl.Wait(context.Background()); err != nil {
		l.Push(lua.LString(err.Error()))

		return 1
	}

	l.Push(lua.LNil)

	return 1
}
