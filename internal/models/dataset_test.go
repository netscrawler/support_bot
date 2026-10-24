package models_test

import (
	"math"
	"testing"

	"support_bot/internal/models"
)

func TestDatasetHash(t *testing.T) {
	base := models.Dataset{
		"s": {{"a": 1, "b": "x"}, {"a": 2, "b": "y"}},
	}
	h := base.Hash()

	if h == "" {
		t.Fatal("Hash() is empty for a marshalable dataset")
	}

	cases := []struct {
		name string
		ds   models.Dataset
		same bool
	}{
		{
			"порядок ключей строки",
			models.Dataset{"s": {{"b": "x", "a": 1}, {"b": "y", "a": 2}}},
			true,
		},
		{"_meta игнорируется", models.Dataset{
			"s":     {{"a": 1, "b": "x"}, {"a": 2, "b": "y"}},
			"_meta": {{"generated": "2026-10-24"}},
		}, true},
		{
			"порядок строк",
			models.Dataset{"s": {{"a": 2, "b": "y"}, {"a": 1, "b": "x"}}},
			false,
		},
		{
			"значения",
			models.Dataset{"s": {{"a": 1, "b": "x"}, {"a": 3, "b": "y"}}},
			false,
		},
	}
	for _, c := range cases {
		if got := c.ds.Hash(); (got == h) != c.same {
			t.Errorf("%s: match = %v, want %v", c.name, got == h, c.same)
		}
	}
}

func TestDatasetHash_Unmarshalable(t *testing.T) {
	if h := (models.Dataset{"s": {{"a": math.NaN()}}}).Hash(); h != "" {
		t.Errorf("Hash() = %q, want empty string", h)
	}
}
