package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/bashfulrobot/snowstorm/internal/query"
)

// accountColumnRe matches a column that identifies the Snowflake account a
// row belongs to (ACCOUNT, ACCOUNT_NAME, AccountName, ...). When a result
// has one, both table and JSON rendering fold repeated account values into
// a single group instead of repeating it on every row.
var accountColumnRe = regexp.MustCompile(`(?i)^account(_?name)?$`)

func findAccountColumn(columns []string) (string, bool) {
	for _, c := range columns {
		if accountColumnRe.MatchString(c) {
			return c, true
		}
	}
	return "", false
}

// openOutput resolves the --out flag to a writer. "" and "-" mean stdout.
// The returned closer is always non-nil and safe to defer-close.
func openOutput(path string) (io.Writer, io.Closer, error) {
	if path == "" || path == "-" {
		return os.Stdout, io.NopCloser(nil), nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open output %q: %w", path, err)
	}
	return f, f, nil
}

// writeResult renders a query.Result as either JSON (default, machine-first)
// or a human-readable table. human only affects the table format (number
// abbreviation/comma-grouping); JSON is always emitted as-is except for the
// account grouping described on findAccountColumn, which only applies when
// groupAccounts is set (the query command's arbitrary, possibly multi-account
// result sets) -- not fixed single-row commands like ping that happen to
// have their own "account" column.
func writeResult(w io.Writer, res *query.Result, format string, human, groupAccounts bool) error {
	switch strings.ToLower(format) {
	case "", "json":
		return writeJSONResult(w, res, groupAccounts)
	case "table":
		return writeTable(w, res, human, groupAccounts)
	default:
		return fmt.Errorf("unknown --format %q (want json or table)", format)
	}
}

// writeJSON pretty-prints any JSON-marshalable value (used by commands like
// discover whose output shape isn't a plain query.Result).
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeJSONResult(w io.Writer, res *query.Result, groupAccounts bool) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if groupAccounts {
		if col, ok := findAccountColumn(res.Columns); ok {
			return enc.Encode(groupByAccount(res, col))
		}
	}
	return enc.Encode(res)
}

// accountGroup is one account's rows, with the account column itself lifted
// out into Account rather than repeated on every row.
type accountGroup struct {
	Account  string           `json:"account"`
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"row_count"`
}

type groupedResult struct {
	Columns  []string       `json:"columns"`
	Accounts []accountGroup `json:"accounts"`
	RowCount int            `json:"row_count"`
}

func groupByAccount(res *query.Result, col string) groupedResult {
	remaining := make([]string, 0, len(res.Columns))
	for _, c := range res.Columns {
		if c != col {
			remaining = append(remaining, c)
		}
	}

	order := make([]string, 0)
	groups := make(map[string]*accountGroup)
	for _, row := range res.Rows {
		key := cellString(row[col], col, false)
		g, ok := groups[key]
		if !ok {
			g = &accountGroup{Account: key, Rows: make([]map[string]any, 0)}
			groups[key] = g
			order = append(order, key)
		}
		stripped := make(map[string]any, len(row))
		for k, v := range row {
			if k == col {
				continue
			}
			stripped[k] = v
		}
		g.Rows = append(g.Rows, stripped)
		g.RowCount++
	}

	out := groupedResult{Columns: remaining, RowCount: res.RowCount}
	for _, key := range order {
		out.Accounts = append(out.Accounts, *groups[key])
	}
	return out
}

func writeTable(w io.Writer, res *query.Result, human, groupAccounts bool) error {
	if groupAccounts {
		if col, ok := findAccountColumn(res.Columns); ok {
			return writeGroupedTable(w, res, col, human)
		}
	}
	return writeFlatTable(w, res.Columns, res.Rows, human, res.RowCount)
}

func writeFlatTable(w io.Writer, columns []string, rows []map[string]any, human bool, rowCount int) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(columns, "\t"))
	// A row of dashes sized to each header's rune length, run through the
	// same tabwriter as every other row so its columns pad/align exactly
	// like the data below it -- no manual width math against tabwriter's
	// own padding rules.
	sep := make([]string, len(columns))
	for i, col := range columns {
		sep[i] = strings.Repeat("-", len([]rune(col)))
	}
	fmt.Fprintln(tw, strings.Join(sep, "\t"))
	for _, row := range rows {
		cells := make([]string, len(columns))
		for i, col := range columns {
			cells[i] = cellString(row[col], col, human)
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if rowCount == 0 {
		fmt.Fprintln(w, "(no rows returned)")
		return nil
	}
	fmt.Fprintf(w, "(%d row(s))\n", rowCount)
	return nil
}

// writeGroupedTable prints one sub-table per distinct account value instead
// of repeating the account column on every row.
func writeGroupedTable(w io.Writer, res *query.Result, col string, human bool) error {
	remaining := make([]string, 0, len(res.Columns))
	for _, c := range res.Columns {
		if c != col {
			remaining = append(remaining, c)
		}
	}

	order := make([]string, 0)
	grouped := make(map[string][]map[string]any)
	for _, row := range res.Rows {
		key := cellString(row[col], col, false)
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], row)
	}

	for i, key := range order {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s: %s\n", col, key)
		rows := grouped[key]
		if err := writeFlatTable(w, remaining, rows, human, len(rows)); err != nil {
			return err
		}
	}
	if res.RowCount == 0 {
		fmt.Fprintln(w, "(no rows returned)")
		return nil
	}
	fmt.Fprintf(w, "\n(%d row(s) total, %d account(s))\n", res.RowCount, len(order))
	return nil
}

func cellString(v any, colName string, human bool) string {
	if v == nil {
		return "-"
	}
	if f, ok := toFloat(v); ok {
		if human && f > 0 && f < 1 && isRatioColumn(colName) {
			return percentString(f)
		}
		if s, ok := formatNumber(f, human); ok {
			return s
		}
	}
	switch t := v.(type) {
	case string:
		s := t
		if human {
			s = shortenTimestamp(s)
			s = truncateLong(s)
		}
		return s
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}

// isRatioColumn reports whether colName has RATE, RATIO, PERCENT, or PCT as
// one of its underscore-delimited tokens (case-insensitive), e.g.
// "UTILIZATION_RATE_LIFETIME" and "EXCHANGE_RATE" both match on the "RATE"
// token. This is deliberately loose (a substring match by token, not by
// meaning) -- the value-range gate in cellString (strictly between 0 and 1)
// is what actually keeps false positives like EXCHANGE_RATE=5 from being
// rewritten as a percentage.
func isRatioColumn(colName string) bool {
	for tok := range strings.SplitSeq(strings.ToUpper(colName), "_") {
		switch tok {
		case "RATE", "RATIO", "PERCENT", "PCT":
			return true
		}
	}
	return false
}

// percentString renders a 0..1 fraction as a percentage, e.g. 0.258 -> "25.8%".
func percentString(f float64) string {
	return trimTrailingZero(f*100) + "%"
}

// timestampRe matches an RFC3339Nano-shaped string: date, "T", time, then an
// optional fractional-second part and a "Z" or numeric offset. Anything that
// doesn't fit this shape (e.g. a bare "2026-07") is left untouched.
var timestampRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)

// shortenTimestamp drops seconds/subseconds and the T/Z decoration from an
// RFC3339Nano-shaped timestamp, e.g. "2026-07-29T12:00:00.000000000Z" ->
// "2026-07-29 12:00". Strings that don't match the shape pass through as-is.
func shortenTimestamp(s string) string {
	if !timestampRe.MatchString(s) {
		return s
	}
	return s[:10] + " " + s[11:16]
}

// maxCellRunes bounds how long a string cell is allowed to render in
// --human table output before it gets truncated with an ellipsis, so a
// single long COMMENT-style value doesn't blow out the whole table's
// column widths.
const maxCellRunes = 70

func truncateLong(s string) string {
	r := []rune(s)
	if len(r) <= maxCellRunes {
		return s
	}
	return string(r[:maxCellRunes]) + "..."
}

// formatNumber renders large numeric values the way a person reading a table
// wants them: comma-grouped from 1,000 up, always (12,345,678,901 stays
// exact -- no flag needed). With human set, it additionally abbreviates from
// 1,000,000 up with a K/M/B/T suffix (5B, 1.2M) for a quick-scan view; that
// rounding is opt-in since it trades precision for brevity. Values under
// 1,000, or values that aren't numeric, fall through to normal formatting
// (ok=false) so exact decimals like 0.258 print untouched.
func formatNumber(f float64, human bool) (string, bool) {
	neg := f < 0
	af := math.Abs(f)

	var s string
	switch {
	case human && af >= 1e12:
		s = trimTrailingZero(af/1e12) + "T"
	case human && af >= 1e9:
		s = trimTrailingZero(af/1e9) + "B"
	case human && af >= 1e6:
		s = trimTrailingZero(af/1e6) + "M"
	case af >= 1000:
		s = commaGroup(af)
	default:
		return "", false
	}
	if neg {
		s = "-" + s
	}
	return s, true
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	default:
		return 0, false
	}
}

func trimTrailingZero(f float64) string {
	s := strconv.FormatFloat(f, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

func commaGroup(f float64) string {
	whole := int64(f)
	out := groupThousands(strconv.FormatInt(whole, 10))
	if frac := f - float64(whole); frac != 0 {
		fs := strconv.FormatFloat(frac, 'f', 2, 64) // "0.42"
		out += strings.TrimPrefix(fs, "0")
	}
	return out
}

func groupThousands(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	lead := n % 3
	if lead > 0 {
		b.WriteString(s[:lead])
		b.WriteByte(',')
	}
	for i := lead; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte(',')
		}
	}
	return b.String()
}
