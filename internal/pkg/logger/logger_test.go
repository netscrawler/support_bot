package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetup_ExcludeFields(t *testing.T) {
	tests := []struct {
		name             string
		config           LogConfig
		expectedExcluded []string
	}{
		{
			name: "default should have all fields",
			config: LogConfig{
				Level:  "debug",
				Format: "text",
				Output: []string{"stdout"},
			},
			expectedExcluded: nil,
		},
		{
			name: "exclude time",
			config: LogConfig{
				Level:   "debug",
				Format:  "text",
				Output:  []string{"stdout"},
				Exclude: []string{"time"},
			},
			expectedExcluded: []string{"time"},
		},
		{
			name: "exclude level and msg",
			config: LogConfig{
				Level:   "debug",
				Format:  "text",
				Output:  []string{"stdout"},
				Exclude: []string{"level", "msg"},
			},
			expectedExcluded: []string{"level", "msg"},
		},
		{
			name: "json format: exclude time",
			config: LogConfig{
				Level:   "debug",
				Format:  "json",
				Output:  []string{"stdout"},
				Exclude: []string{"time"},
			},
			expectedExcluded: []string{"time"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := getOpts(tt.config)

			log := getLogger(tt.config.Format, &buf, opts)
			log.Info("test message")

			output := buf.String()

			for _, field := range []string{"time", "level", "msg"} {
				excluded := false
				for _, e := range tt.expectedExcluded {
					if e == field {
						excluded = true
						break
					}
				}

				var found bool
				if tt.config.Format == "json" {
					found = strings.Contains(output, "\""+field+"\":")
				} else {
					found = strings.Contains(output, field+"=")
				}

				if excluded && found {
					t.Errorf("field %s should be excluded but found in output: %s", field, output)
				}
				if !excluded && !found {
					t.Errorf("field %s should be present but not found in output: %s", field, output)
				}
			}
		})
	}
}
