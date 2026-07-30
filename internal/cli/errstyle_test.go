package cli

import (
	"errors"
	"testing"
)

func TestShouldColorizeError(t *testing.T) {
	cases := []struct {
		name        string
		noColorFlag bool
		noColorEnv  string
		stderrIsTTY bool
		want        bool
	}{
		{"real terminal, no --no-color, no NO_COLOR: colorize", false, "", true, true},
		{"--no-color set: never colorize even at a real terminal", true, "", true, false},
		{"NO_COLOR set (any non-empty value): never colorize", false, "1", true, false},
		{"stderr not a terminal: never colorize even with neither flag/env set", false, "", false, false},
		{"both --no-color and NO_COLOR set: never colorize", true, "1", true, false},
		{"NO_COLOR set to empty string counts as unset per no-color.org", false, "", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldColorizeError(tc.noColorFlag, tc.noColorEnv, tc.stderrIsTTY)
			if got != tc.want {
				t.Fatalf("shouldColorizeError(%v, %q, %v) = %v, want %v",
					tc.noColorFlag, tc.noColorEnv, tc.stderrIsTTY, got, tc.want)
			}
		})
	}
}

func TestPrintErrorDoesNotPanic(t *testing.T) {
	// Smoke check only -- PrintError writes straight to os.Stderr and has no
	// return value to assert on; the real coverage is shouldColorizeError.
	PrintError(nil)
	PrintError(errors.New("boom"))
}
