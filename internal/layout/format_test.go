package layout

import "testing"

func TestFormatValue(t *testing.T) {
	cases := []struct {
		name   string
		value  any
		format string
		want   string
	}{
		{"nil default", nil, "", ""},
		{"nil percent", nil, "percent", ""},
		{"integral float default", 140.0, "", "140"},
		{"fraction default", 4.6, "", "4.6"},
		{"fraction trims zero", 4.10, "", "4.1"},
		{"string default", "web", "", "web"},
		{"int number", 994, "number", "994"},
		{"grouped number", 12345, "number", "12 345"},
		{"grouped number millions", 1234567, "number", "1 234 567"},
		{"float number rounds", 138.7, "number", "139"},
		{"percent one decimal", 0.895, "percent", "89.5%"},
		{"percent trims zero", 0.46, "percent", "46%"},
		{"percent whole", 1.0, "percent", "100%"},
		{"money", 1234.5, "money", "1 234.50"},
		{"money negative", -1234.5, "money", "-1 234.50"},
		{"unknown format falls back", 4.6, "weird", "4.6"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatValue(tc.value, tc.format); got != tc.want {
				t.Fatalf("FormatValue(%v, %q) = %q, want %q", tc.value, tc.format, got, tc.want)
			}
		})
	}
}
