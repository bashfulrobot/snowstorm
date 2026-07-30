package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/bashfulrobot/snowstorm/internal/query"
)

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
// or a human-readable table.
func writeResult(w io.Writer, res *query.Result, format string) error {
	switch strings.ToLower(format) {
	case "", "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	case "table":
		return writeTable(w, res)
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

func writeTable(w io.Writer, res *query.Result) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(res.Columns, "\t"))
	for _, row := range res.Rows {
		cells := make([]string, len(res.Columns))
		for i, col := range res.Columns {
			cells[i] = cellString(row[col])
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(w, "(%d row(s))\n", res.RowCount)
	return nil
}

func cellString(v any) string {
	if v == nil {
		return "NULL"
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
