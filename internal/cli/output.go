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
		key := cellString(row[col], false)
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
	for _, row := range rows {
		cells := make([]string, len(columns))
		for i, col := range columns {
			cells[i] = cellString(row[col], human)
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
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
		key := cellString(row[col], false)
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
	fmt.Fprintf(w, "\n(%d row(s) total, %d account(s))\n", res.RowCount, len(order))
	return nil
}

func cellString(v any, human bool) string {
	if v == nil {
		return "NULL"
	}
	if human {
		if s, ok := humanNumber(v); ok {
			return s
		}
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(b)
	}
}

// humanNumber renders large numeric values the way a person reading a table
// wants them: comma-grouped from 1,000 up, abbreviated with a K/M/B/T suffix
// from 1,000,000 up (e.g. 5B, 12,345, 1.2M). Values under 1,000, or values
// that aren't numeric, fall through to normal formatting.
func humanNumber(v any) (string, bool) {
	f, ok := toFloat(v)
	if !ok {
		return "", false
	}
	if math.Abs(f) < 1000 {
		return "", false
	}
	return formatHuman(f), true
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

func formatHuman(f float64) string {
	neg := f < 0
	af := math.Abs(f)

	var s string
	switch {
	case af >= 1e12:
		s = trimTrailingZero(af/1e12) + "T"
	case af >= 1e9:
		s = trimTrailingZero(af/1e9) + "B"
	case af >= 1e6:
		s = trimTrailingZero(af/1e6) + "M"
	default: // 1,000 <= af < 1,000,000
		s = commaGroup(af)
	}
	if neg {
		s = "-" + s
	}
	return s
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
