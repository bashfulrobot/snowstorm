package cli

import (
	"bytes"
	"encoding/json"
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
	if got := cellString(int64(3300000000), "ROW_COUNT", false); got != "3,300,000,000" {
		t.Errorf("cellString(3300000000, false) = %q, want %q (commas by default)", got, "3,300,000,000")
	}
	if got := cellString(int64(4200), "ROW_COUNT", true); got != "4,200" {
		t.Errorf("cellString(4200, true) = %q, want %q", got, "4,200")
	}
	if got := cellString(int64(500), "ROW_COUNT", true); got != "500" {
		t.Errorf("cellString(500, true) = %q, want %q (below 1000 stays plain)", got, "500")
	}
	if got := cellString(int64(4200), "ROW_COUNT", false); got != "4,200" {
		t.Errorf("cellString(4200, false) = %q, want %q (no --human, commas still apply)", got, "4,200")
	}
	if got := cellString(0.258, "SOME_VALUE", false); got != "0.258" {
		t.Errorf("cellString(0.258, false) = %q, want %q (small decimals stay exact, untouched)", got, "0.258")
	}
	if got := cellString(nil, "ROW_COUNT", true); got != "-" {
		t.Errorf("cellString(nil, true) = %q, want %q (NULL renders as a dash)", got, "-")
	}
}

func TestCellStringNullIsDash(t *testing.T) {
	// Unconditional (no --human needed): NULL cells render as a single dash,
	// not the literal string "NULL".
	if got := cellString(nil, "ANY_COLUMN", false); got != "-" {
		t.Errorf("cellString(nil, false) = %q, want %q", got, "-")
	}
	if got := cellString(nil, "ANY_COLUMN", true); got != "-" {
		t.Errorf("cellString(nil, true) = %q, want %q", got, "-")
	}
}

func TestCellStringRatioToPercent(t *testing.T) {
	cases := []struct {
		name  string
		v     any
		col   string
		human bool
		want  string
	}{
		{"true positive: RATE token, value in (0,1), human on", 0.258, "UTILIZATION_RATE_LIFETIME", true, "25.8%"},
		{"true positive: exact half", 0.5, "SUCCESS_RATIO", true, "50%"},
		{"not converted: value outside (0,1) even with RATE column", 5.0, "EXCHANGE_RATE", true, "5"},
		{"not converted: non-ratio column name", 0.5, "SOME_VALUE", true, "0.5"},
		{"not converted: human off, even with matching name+range", 0.258, "UTILIZATION_RATE_LIFETIME", false, "0.258"},
		{"not converted: value exactly 0", 0.0, "SUCCESS_RATE", true, "0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cellString(c.v, c.col, c.human)
			if got != c.want {
				t.Errorf("cellString(%v, %q, %v) = %q, want %q", c.v, c.col, c.human, got, c.want)
			}
		})
	}
}

func TestCellStringTimestampShortening(t *testing.T) {
	cases := []struct {
		in    string
		human bool
		want  string
	}{
		{"2026-07-29T12:00:00.000000000Z", true, "2026-07-29 12:00"},
		{"2026-07-29T12:00:00Z", true, "2026-07-29 12:00"},
		{"2026-07-29T12:00:00.123456+05:00", true, "2026-07-29 12:00"},
		{"2026-07-29T12:00:00.000000000Z", false, "2026-07-29T12:00:00.000000000Z"}, // not gated without --human
		{"2026-07", true, "2026-07"},                                                // doesn't match the shape, left untouched
		{"just a string", true, "just a string"},
	}
	for _, c := range cases {
		got := cellString(c.in, "CREATED_AT", c.human)
		if got != c.want {
			t.Errorf("cellString(%q, CREATED_AT, %v) = %q, want %q", c.in, c.human, got, c.want)
		}
	}
}

func TestCellStringLongStringTruncation(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := cellString(long, "COMMENT", true)
	wantLen := maxCellRunes + len("...")
	if len([]rune(got)) != wantLen {
		t.Errorf("cellString(long comment, true) length = %d, want %d", len([]rune(got)), wantLen)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("cellString(long comment, true) = %q, want trailing ellipsis", got)
	}

	// Without --human, no truncation.
	if got := cellString(long, "COMMENT", false); got != long {
		t.Errorf("cellString(long comment, false) should be untruncated, got len %d", len(got))
	}

	// A string at or under the limit is untouched.
	short := strings.Repeat("b", maxCellRunes)
	if got := cellString(short, "COMMENT", true); got != short {
		t.Errorf("cellString(short comment, true) should be untruncated, got %q", got)
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

func TestWriteFlatTableHasSeparatorLine(t *testing.T) {
	res := &query.Result{
		Columns:  []string{"TABLE_NAME", "ROW_COUNT"},
		Rows:     []map[string]any{{"TABLE_NAME": "ORDERS", "ROW_COUNT": int64(42)}},
		RowCount: 1,
	}
	var buf bytes.Buffer
	if err := writeTable(&buf, res, false, false); err != nil {
		t.Fatalf("writeTable: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected header, separator, and data lines, got:\n%s", buf.String())
	}
	header := lines[0]
	sep := lines[1]
	if !strings.Contains(sep, "-") {
		t.Errorf("expected a dash separator line after the header, got %q", sep)
	}
	if strings.ContainsAny(sep, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Errorf("separator line should contain only dashes/whitespace, got %q", sep)
	}
	// Roughly the same rendered width as the header (tabwriter pads both
	// consistently, so they should line up).
	if len(sep) < len(header)-2 || len(sep) > len(header)+2 {
		t.Errorf("separator line width %d should roughly match header width %d\nheader: %q\nsep:    %q", len(sep), len(header), header, sep)
	}
}

func TestWriteFlatTableEmptyResultMessage(t *testing.T) {
	res := &query.Result{
		Columns:  []string{"TABLE_NAME", "ROW_COUNT"},
		Rows:     []map[string]any{},
		RowCount: 0,
	}
	var buf bytes.Buffer
	if err := writeTable(&buf, res, false, false); err != nil {
		t.Fatalf("writeTable: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(no rows returned)") {
		t.Errorf("expected \"(no rows returned)\" for zero rows, got:\n%s", out)
	}
	if strings.Contains(out, "(0 row(s))") {
		t.Errorf("should not fall back to the old \"(0 row(s))\" message, got:\n%s", out)
	}

	// Nonzero counts keep the existing message.
	res.Rows = []map[string]any{{"TABLE_NAME": "ORDERS", "ROW_COUNT": int64(1)}}
	res.RowCount = 1
	buf.Reset()
	if err := writeTable(&buf, res, false, false); err != nil {
		t.Fatalf("writeTable: %v", err)
	}
	if !strings.Contains(buf.String(), "(1 row(s))") {
		t.Errorf("expected \"(1 row(s))\" for nonzero rows, got:\n%s", buf.String())
	}
}

func TestWriteGroupedTableEmptyResultMessage(t *testing.T) {
	res := &query.Result{
		Columns:  []string{"ACCOUNT_NAME", "TABLE_NAME"},
		Rows:     []map[string]any{},
		RowCount: 0,
	}
	var buf bytes.Buffer
	if err := writeTable(&buf, res, false, true); err != nil {
		t.Fatalf("writeTable: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(no rows returned)") {
		t.Errorf("expected \"(no rows returned)\" for zero rows in grouped table, got:\n%s", out)
	}
	if strings.Contains(out, "row(s) total") {
		t.Errorf("should not fall back to the old totals message, got:\n%s", out)
	}
}

func TestJSONResultUnaffectedByTableFormatting(t *testing.T) {
	// --format json must stay byte-for-byte exact regardless of any table
	// formatting behavior (NULL-as-dash, percent conversion, timestamp
	// shortening, truncation, comma-grouping, etc.) -- none of it applies here.
	res := &query.Result{
		Columns: []string{"UTILIZATION_RATE_LIFETIME", "CREATED_AT", "COMMENT", "MAYBE_NULL", "ROW_COUNT"},
		Rows: []map[string]any{
			{
				"UTILIZATION_RATE_LIFETIME": 0.258,
				"CREATED_AT":                "2026-07-29T12:00:00.000000000Z",
				"COMMENT":                   strings.Repeat("a", 100),
				"MAYBE_NULL":                nil,
				"ROW_COUNT":                 int64(3300000000),
			},
		},
		RowCount: 1,
	}

	var wantBuf bytes.Buffer
	encWant := json.NewEncoder(&wantBuf)
	encWant.SetIndent("", "  ")
	if err := encWant.Encode(res); err != nil {
		t.Fatalf("encode expected: %v", err)
	}

	var gotBuf bytes.Buffer
	if err := writeResult(&gotBuf, res, "json", true, false); err != nil {
		t.Fatalf("writeResult json: %v", err)
	}

	if gotBuf.String() != wantBuf.String() {
		t.Errorf("JSON output changed by table-formatting logic.\ngot:\n%s\nwant:\n%s", gotBuf.String(), wantBuf.String())
	}
	if strings.Contains(gotBuf.String(), "25.8%") {
		t.Errorf("JSON output must not apply percent conversion, got:\n%s", gotBuf.String())
	}
	if strings.Contains(gotBuf.String(), "...") {
		t.Errorf("JSON output must not apply long-string truncation, got:\n%s", gotBuf.String())
	}
	if strings.Contains(gotBuf.String(), "2026-07-29 12:00") {
		t.Errorf("JSON output must not apply timestamp shortening, got:\n%s", gotBuf.String())
	}
	if strings.Contains(gotBuf.String(), `"-"`) {
		t.Errorf("JSON output must not render NULL as a dash, got:\n%s", gotBuf.String())
	}
}
