package secret

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const real = "s3cr3t-value"

type inner struct {
	Token String `json:"token" yaml:"token"`
}

type outer struct {
	Key   String `json:"key"   yaml:"key"`
	Inner inner  `json:"inner" yaml:"inner"`
}

func sample() outer { return outer{Key: real, Inner: inner{Token: real}} }

func TestSlog(t *testing.T) {
	for name, mk := range map[string]func(*bytes.Buffer) slog.Handler{
		"json": func(b *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(b, nil) },
		"text": func(b *bytes.Buffer) slog.Handler { return slog.NewTextHandler(b, nil) },
	} {
		var buf bytes.Buffer
		slog.New(mk(&buf)).Info("x", "cfg", sample(), "p", sample().Key)
		out := buf.String()
		if strings.Contains(out, real) || !strings.Contains(out, mask) {
			t.Errorf("%s: %s", name, out)
		}
	}
}

func TestFmt(t *testing.T) {
	for _, v := range []any{sample(), &[]outer{sample()}[0], sample().Key} {
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			if out := fmt.Sprintf(verb, v); strings.Contains(out, real) {
				t.Errorf("%s: %s", verb, out)
			}
		}
	}
}

func TestJSONYAMLValue(t *testing.T) {
	b, err := json.Marshal(sample())
	if err != nil || strings.Contains(string(b), real) || !strings.Contains(string(b), mask) {
		t.Errorf("json: %s %v", b, err)
	}
	y, err := yaml.Marshal(sample())
	if err != nil || !strings.Contains(string(y), real) {
		t.Errorf("yaml: %s %v", y, err)
	}
	if sample().Key.Value() != real {
		t.Error("Value")
	}
}

func TestRedact(t *testing.T) {
	reset()
	t.Cleanup(reset)
	Register("abcd", "xxabcdyy", "abc", "", "abcd")
	if got := Redact("a xxabcdyy b abcd c abc"); got != "a *** b *** c abc" {
		t.Errorf("got %q", got)
	}
	if len(secrets) != 2 {
		t.Errorf("secrets: %v", secrets)
	}
}
