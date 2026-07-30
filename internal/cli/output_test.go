package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bashfulrobot/snowstorm/internal/query"
)

func TestFormatNumberDefault(t *testing.T) {
	// human=false: always comma-group from 1,000 up, never abbreviate.
	cases := []struct {
		in   float64
		want string
		ok   bool
	}{
		{999, "", false},
		{1000, "1,000", true},
		{12345, "12,345", true},
		{999999, "999,999", true},
		{1000000, "1,000,000", true},
		{5234000000, "5,234,000,000", true},
		{12345678901, "12,345,678,901", true},
		{-1500, "-1,500", true},
		{-2500000, "-2,500,000", true},
	}
	for _, c := range cases {
		got, ok := formatNumber(c.in, false)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("formatNumber(%v, false) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestFormatNumberHuman(t *testing.T) {
	// human=true: comma-group 1,000-1M, abbreviate with K/M/B/T from 1M up.
	cases := []struct {
		in   float64
		want string
		ok   bool
	}{
		{999, "", false},
		{1000, "1,000", true},
		{12345, "12,345", true},
		{999999, "999,999", true},
		{1000000, "1M", true},
		{1234567, "1.2M", true},
		{5000000000, "5B", true},
		{5234000000, "5.2B", true},
		{12345678901, "12.3B", true},
		{1000000000000, "1T", true},
		{-1500, "-1,500", true},
		{-2500000, "-2.5M", true},
	}
	for _, c := range cases {
		got, ok := formatNumber(c.in, true)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("formatNumber(%v, true) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestCellStringNumbers(t *testing.T) {
	// Commas apply by default (no --human needed) -- this was the reported gap:
	// large numbers in plain `--format table` output were unreadable digit soup.
	if got := cellString(int64(3300000000), false); got != "3,300,000,000" {
		t.Errorf("cellString(3300000000, false) = %q, want %q (commas by default)", got, "3,300,000,000")
	}
	if got := cellString(int64(4200), true); got != "4,200" {
		t.Errorf("cellString(4200, true) = %q, want %q", got, "4,200")
	}
	if got := cellString(int64(500), true); got != "500" {
		t.Errorf("cellString(500, true) = %q, want %q (below 1000 stays plain)", got, "500")
	}
	if got := cellString(int64(4200), false); got != "4,200" {
		t.Errorf("cellString(4200, false) = %q, want %q (no --human, commas still apply)", got, "4,200")
	}
	if got := cellString(0.258, false); got != "0.258" {
		t.Errorf("cellString(0.258, false) = %q, want %q (small decimals stay exact, untouched)", got, "0.258")
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
