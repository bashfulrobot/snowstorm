package cli

import "testing"

func TestShouldShowPicker(t *testing.T) {
	cases := []struct {
		name                                string
		hasArg, hasFile, hasSaved, skipPick bool
		stdinIsTTY, stdoutIsTTY             bool
		want                                bool
	}{
		{"bare invocation at a real terminal: picker shows", false, false, false, false, true, true, true},
		{"SQL arg given: never shows, regardless of TTY", true, false, false, false, true, true, false},
		{"--file given: never shows", false, true, false, false, true, true, false},
		{"--saved given: never shows", false, false, true, false, true, true, false},
		{"--skip-pick given at a real terminal: never shows", false, false, false, true, true, true, false},
		{"piped stdin: never shows even without --skip-pick", false, false, false, false, false, true, false},
		{"piped stdout: never shows even without --skip-pick", false, false, false, false, true, false, false},
		{"both piped: never shows", false, false, false, false, false, false, false},
		{"arg AND non-TTY: still false (arg check short-circuits)", true, false, false, false, false, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldShowPicker(tc.hasArg, tc.hasFile, tc.hasSaved, tc.skipPick, tc.stdinIsTTY, tc.stdoutIsTTY)
			if got != tc.want {
				t.Fatalf("shouldShowPicker(%v, %v, %v, %v, %v, %v) = %v, want %v",
					tc.hasArg, tc.hasFile, tc.hasSaved, tc.skipPick, tc.stdinIsTTY, tc.stdoutIsTTY, got, tc.want)
			}
		})
	}
}

func TestQueryItemFilterAndDisplay(t *testing.T) {
	it := queryItem{name: "top_accounts", description: "top accounts by ARR"}
	if it.Title() != "top_accounts" {
		t.Errorf("Title() = %q, want %q", it.Title(), "top_accounts")
	}
	if it.Description() != "top accounts by ARR" {
		t.Errorf("Description() = %q, want %q", it.Description(), "top accounts by ARR")
	}
	if it.FilterValue() != "top_accounts top accounts by ARR" {
		t.Errorf("FilterValue() = %q, want name+description", it.FilterValue())
	}
}
