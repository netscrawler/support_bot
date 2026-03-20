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
		reg.Register("", noopAction{})
	})
}

func TestRegistryRegister_PanicsOnNilAction(t *testing.T) {
	reg := New()

	assertPanics(t, func() {
		reg.Register("std@noop", nil)
	})
}

func TestRegistryRegisterOrReplace_PanicsOnInvalidInput(t *testing.T) {
	reg := New()

	assertPanics(t, func() {
		reg.RegisterOrReplace("   ", noopAction{})
	})

	assertPanics(t, func() {
		reg.RegisterOrReplace("std@noop", nil)
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
