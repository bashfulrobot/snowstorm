package cli

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

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

func TestMatchPrefix(t *testing.T) {
	items := []list.Item{
		queryItem{name: "acme_usage", description: "Acme buy vs use"},
		queryItem{name: "zillow_usage", description: "Zillow buy vs use"},
		queryItem{name: "top_accounts", description: "top accounts by ARR"},
	}
	names := func(got []list.Item) []string {
		var out []string
		for _, it := range got {
			out = append(out, it.(queryItem).name)
		}
		return out
	}

	cases := []struct {
		term string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"ac", []string{"acme_usage"}},
		{"ACME", []string{"acme_usage"}},
		{"usage", nil},
		{"cme", nil},
		{"zillow_usage zillow", []string{"zillow_usage"}},
		{"nope", nil},
	}
	for _, tc := range cases {
		got := names(matchPrefix(items, tc.term))
		if len(got) != len(tc.want) {
			t.Fatalf("matchPrefix(%q) = %v, want %v", tc.term, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("matchPrefix(%q) = %v, want %v", tc.term, got, tc.want)
			}
		}
	}
}

func TestPickerStartsEmptyAndFiltersOnType(t *testing.T) {
	m := newPickerModel([]savedQueryInfo{
		{Name: "acme_usage", Description: "Acme"},
		{Name: "zillow_usage", Description: "Zillow"},
	})
	if n := len(m.list.Items()); n != 0 {
		t.Fatalf("picker opened with %d items listed, want 0", n)
	}
	m.input.SetValue("zi")
	m.list.SetItems(matchPrefix(m.all, m.input.Value()))
	if n := len(m.list.Items()); n != 1 {
		t.Fatalf("after typing %q got %d items, want 1", "zi", n)
	}
}
