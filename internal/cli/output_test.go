package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bashfulrobot/snowstorm/internal/query"
)

func TestFormatHuman(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{999999, "999,999"},
		{1000000, "1M"},
		{1234567, "1.2M"},
		{5000000000, "5B"},
		{5234000000, "5.2B"},
		{12345678901, "12.3B"},
		{1000000000000, "1T"},
		{-1500, "-1,500"},
		{-2500000, "-2.5M"},
	}
	for _, c := range cases {
		if got := formatHuman(c.in); got != c.want {
			t.Errorf("formatHuman(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCellStringHuman(t *testing.T) {
	if got := cellString(int64(4200), true); got != "4,200" {
		t.Errorf("cellString(4200, true) = %q, want %q", got, "4,200")
	}
	if got := cellString(int64(500), true); got != "500" {
		t.Errorf("cellString(500, true) = %q, want %q (below 1000 stays plain)", got, "500")
	}
	if got := cellString(int64(4200), false); got != "4200" {
		t.Errorf("cellString(4200, false) = %q, want %q (no --human, no grouping)", got, "4200")
	}
	if got := cellString(nil, true); got != "NULL" {
		t.Errorf("cellString(nil, true) = %q, want NULL", got)
	}
}

func TestWriteTableGroupsAccountColumn(t *testing.T) {
	res := &query.Result{
		Columns: []string{"ACCOUNT_NAME", "TABLE_NAME", "ROW_COUNT"},
		Rows: []map[string]any{
			{"ACCOUNT_NAME": "acme", "TABLE_NAME": "ORDERS", "ROW_COUNT": int64(5234000000)},
			{"ACCOUNT_NAME": "acme", "TABLE_NAME": "USERS", "ROW_COUNT": int64(4200)},
			{"ACCOUNT_NAME": "globex", "TABLE_NAME": "LOGS", "ROW_COUNT": int64(12345678901)},
		},
		RowCount: 3,
	}

	var buf bytes.Buffer
	if err := writeTable(&buf, res, true, true); err != nil {
		t.Fatalf("writeTable: %v", err)
	}
	out := buf.String()

	if strings.Count(out, "acme") != 1 {
		t.Errorf("expected account value \"acme\" to appear once (grouped), got:\n%s", out)
	}
	if !strings.Contains(out, "5.2B") {
		t.Errorf("expected humanized 5.2B in output, got:\n%s", out)
	}
	if !strings.Contains(out, "4,200") {
		t.Errorf("expected comma-grouped 4,200 in output, got:\n%s", out)
	}
	if strings.Contains(out, "ACCOUNT_NAME\t") {
		t.Errorf("account column should not appear as a table column, got:\n%s", out)
	}
}

func TestWriteTableFlatWithoutAccountColumn(t *testing.T) {
	res := &query.Result{
		Columns:  []string{"TABLE_NAME", "ROW_COUNT"},
		Rows:     []map[string]any{{"TABLE_NAME": "ORDERS", "ROW_COUNT": int64(42)}},
		RowCount: 1,
	}
	var buf bytes.Buffer
	if err := writeTable(&buf, res, false, true); err != nil {
		t.Fatalf("writeTable: %v", err)
	}
	if !strings.Contains(buf.String(), "TABLE_NAME") {
		t.Errorf("expected flat table header, got:\n%s", buf.String())
	}
}

func TestGroupByAccount(t *testing.T) {
	res := &query.Result{
		Columns: []string{"ACCOUNT", "TABLE_NAME"},
		Rows: []map[string]any{
			{"ACCOUNT": "acme", "TABLE_NAME": "A"},
			{"ACCOUNT": "globex", "TABLE_NAME": "B"},
			{"ACCOUNT": "acme", "TABLE_NAME": "C"},
		},
		RowCount: 3,
	}
	grouped := groupByAccount(res, "ACCOUNT")

	if len(grouped.Accounts) != 2 {
		t.Fatalf("expected 2 account groups, got %d", len(grouped.Accounts))
	}
	if grouped.Accounts[0].Account != "acme" || grouped.Accounts[0].RowCount != 2 {
		t.Errorf("expected acme group with 2 rows first (encounter order), got %+v", grouped.Accounts[0])
	}
	for _, g := range grouped.Accounts {
		for _, row := range g.Rows {
			if _, ok := row["ACCOUNT"]; ok {
				t.Errorf("account column should be stripped from row, got %+v", row)
			}
		}
	}
	for _, c := range grouped.Columns {
		if c == "ACCOUNT" {
			t.Errorf("ACCOUNT should not appear in remaining Columns: %v", grouped.Columns)
		}
	}
}

func TestFindAccountColumn(t *testing.T) {
	cases := []struct {
		cols []string
		want string
		ok   bool
	}{
		{[]string{"ACCOUNT_NAME", "X"}, "ACCOUNT_NAME", true},
		{[]string{"account", "X"}, "account", true},
		{[]string{"AccountName", "X"}, "AccountName", true},
		{[]string{"TABLE_NAME", "ROW_COUNT"}, "", false},
		{[]string{"ACCOUNT_ID"}, "", false}, // deliberately not matched -- not the display name
	}
	for _, c := range cases {
		got, ok := findAccountColumn(c.cols)
		if ok != c.ok || got != c.want {
			t.Errorf("findAccountColumn(%v) = (%q, %v), want (%q, %v)", c.cols, got, ok, c.want, c.ok)
		}
	}
}
