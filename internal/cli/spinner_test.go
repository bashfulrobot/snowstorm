package cli

import "testing"

func TestShouldShowSpinner(t *testing.T) {
	cases := []struct {
		name        string
		format      string
		skipSpinner bool
		stderrIsTTY bool
		want        bool
	}{
		{"table format, real stderr terminal: spinner shows", "table", false, true, true},
		{"table format, case-insensitive", "TABLE", false, true, true},
		{"json format: never shows even at a real terminal", "json", false, true, false},
		{"table format, --skip-spinner set: never shows", "table", true, true, false},
		{"table format, stderr not a terminal: never shows", "table", false, false, false},
		{"table format, --skip-spinner AND non-tty: never shows", "table", true, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldShowSpinner(tc.format, tc.skipSpinner, tc.stderrIsTTY)
			if got != tc.want {
				t.Fatalf("shouldShowSpinner(%q, %v, %v) = %v, want %v",
					tc.format, tc.skipSpinner, tc.stderrIsTTY, got, tc.want)
			}
		})
	}
}
