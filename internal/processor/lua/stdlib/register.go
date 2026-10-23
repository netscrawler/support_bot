package stdlib

import luaEngine "github.com/yuin/gopher-lua"

type Plug interface {
	Register(l *luaEngine.LState)
}

type STD struct {
	plugs []Plug
}

func NewSTD(plugs ...Plug) *STD {
	return &STD{
		plugs: plugs,
	}
}

func (s *STD) Register(l *luaEngine.LState) {
	for _, plug := range s.plugs {
		plug.Register(l)
	}
}
