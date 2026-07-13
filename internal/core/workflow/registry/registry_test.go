package registry

import (
	"context"
	"testing"
)

type noopAction struct{}

func (noopAction) Execute(context.Context, ActionInput) (ActionOutput, error) {
	return ActionOutput{}, nil
}

func TestRegistryRegister_PanicsOnEmptyType(t *testing.T) {
	reg := New()

	assertPanics(t, func() {
		err := reg.Register("", noopAction{})
		if err != nil {
			panic(err)
		}
	})
}

func TestRegistryRegister_PanicsOnNilAction(t *testing.T) {
	reg := New()

	assertPanics(t, func() {
		err := reg.Register("std@noop", nil)
		if err != nil {
			panic(err)
		}
	})
}

func TestRegistryRegisterOrReplace_PanicsOnInvalidInput(t *testing.T) {
	reg := New()

	assertPanics(t, func() {
		err := reg.RegisterOrReplace("   ", noopAction{})
		if err != nil {
			panic(err)
		}
	})

	assertPanics(t, func() {
		err := reg.RegisterOrReplace("std@noop", nil)
		if err != nil {
			panic(err)
		}
	})
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic, got none")
		}
	}()

	fn()
}
